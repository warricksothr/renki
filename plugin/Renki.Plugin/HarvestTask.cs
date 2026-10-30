using System;
using System.Collections.Generic;
using System.Linq;
using System.Reflection;
using HarmonyLib;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One resource, using the same held primary-attack input as
    /// a player. Descendant logs/stumps remain separate controller targets.</summary>
    sealed class HarvestTask : IBodyTask
    {
        internal static HarvestTask Active;
        readonly HarvestArgs args;
        readonly GameObject target;
        readonly ZNetView view;
        readonly Collider[] targetColliders;
        readonly HashSet<string> yields = new HashSet<string>();
        readonly List<Character> nearbyCharacters = new List<Character>();
        readonly HashSet<ZDOID> oldDrops = new HashSet<ZDOID>();
        readonly Dictionary<string, int> before = new Dictionary<string, int>();
        readonly Collider[] hits = new Collider[256];
        readonly Dictionary<TreeLog, HarvestStillness> logPoses = new Dictionary<TreeLog, HarvestStillness>();
        readonly HarvestEvidence evidence = new HarvestEvidence();
        readonly Walker walker = new Walker();
        readonly int minimum;
        readonly bool rock, tallTree, rolling;
        readonly string prefab;
        readonly float startedAt = Time.unscaledTime;
        readonly float initialHealth;
        float nextThreatCheck;
        ItemDrop.ItemData tool;
        Vector3 origin, walkGoal, retreat;
        bool walking, retreatReady, retreated, broken, cancelled, injured, forceApproach;
        float brokeAt, quietSince = -1, nextCollect, swingSince = -1;
        int hitsAtSwingStart, collected, contactRetries;
        PickupTask pickup;

        public HarvestTask(HarvestArgs args, GameObject target)
        {
            this.args = args; this.target = target;
            if (target == null || (view = target.GetComponent<ZNetView>()) == null || !view.IsValid())
                throw new TaskRefused(Wire.ReasonInvalidTarget, "no loaded harvestable resource at that entity");
            DropTable drops = null;
            if (target.TryGetComponent<TreeBase>(out var tree))
            {
                minimum = tree.m_minToolTier; tallTree = rolling = true; drops = tree.m_dropWhenDestroyed;
            }
            else if (target.TryGetComponent<TreeLog>(out var log))
            {
                minimum = log.m_minToolTier; rolling = true; drops = log.m_dropWhenDestroyed;
            }
            else if (target.TryGetComponent<Destructible>(out var wood) && wood.m_destructibleType == DestructibleType.Tree)
                minimum = wood.m_minToolTier;
            else if (target.TryGetComponent<MineRock>(out var mine))
            {
                minimum = mine.m_minToolTier; rock = true; drops = mine.m_dropItems;
            }
            else if (target.TryGetComponent<MineRock5>(out var mine5))
            {
                minimum = mine5.m_minToolTier; rock = true; drops = mine5.m_dropItems;
            }
            else throw new TaskRefused(Wire.ReasonInvalidTarget, "harvest requires a tree, log, stump or mineable rock");
            origin = target.transform.position;
            prefab = Perceiver.PrefabName(view, target);
            targetColliders = target.GetComponentsInChildren<Collider>();
            if (!targetColliders.Any(Alive)) throw new TaskRefused(Wire.ReasonInvalidTarget, "resource has no active physical target");
            if (!PrivateArea.CheckAccess(origin, 0f, false, false))
                throw new TaskRefused(Wire.ReasonNotAllowed, "the resource is behind a ward");
            if (drops != null)
                foreach (var drop in drops.m_drops)
                    if (drop.m_item != null && drop.m_item.GetComponent<ItemDrop>() != null) yields.Add(drop.m_item.name);
            // Small destructible trees describe drops through DropOnDestroyed.
            foreach (var source in target.GetComponents<DropOnDestroyed>())
                foreach (var drop in source.m_dropWhenDestroyed.m_drops)
                    if (drop.m_item != null && drop.m_item.GetComponent<ItemDrop>() != null) yields.Add(drop.m_item.name);
            var p = Player.m_localPlayer;
            tool = Best(p);
            if (tool == null) throw new TaskRefused(Wire.ReasonNoTool, $"need a usable {(rock ? "pickaxe" : "axe")} of tier {minimum}");
            initialHealth = p.GetHealth();
            if (!HarvestHealth.Ready(initialHealth, p.GetMaxHealth()))
                throw new TaskRefused(Wire.ReasonRefused, "recover to at least 80% health before harvesting");
            if (ThreatNear(p)) throw new TaskRefused(Wire.ReasonRefused, "an enemy is within 15 m; clear the work site before harvesting");
            if (args.Collect)
            {
                foreach (var item in p.GetInventory().GetAllItems())
                {
                    var name = Name(item);
                    before[name] = (before.TryGetValue(name, out var n) ? n : 0) + item.m_stack;
                }
                var nearby = NearbyDrops();
                if (nearby == null) throw new TaskRefused(Wire.ReasonRefused, "too many nearby drops to identify safely");
                foreach (var drop in nearby) oldDrops.Add(drop.m_nview.GetZDO().m_uid);
            }
            Active = this;
            Tracer.Emit("harvest", "starting resource", new Dictionary<string, object>
            {
                ["entity"] = args.Entity,
                ["prefab"] = prefab,
                ["tool"] = Name(tool),
                ["minimum"] = minimum,
            });
        }

        static string Name(ItemDrop.ItemData item) => item.m_dropPrefab != null ? item.m_dropPrefab.name : item.m_shared.m_name;
        static bool Alive(Collider c) => c != null && c.enabled && !c.isTrigger && c.gameObject.activeInHierarchy;
        ItemDrop.ItemData Best(Player p) => HarvestTool.Best(p.GetInventory().GetAllItems(), minimum,
            item => item.m_shared.m_toolTier, item => rock ? item.GetDamage().m_pickaxe : item.GetDamage().m_chop,
            item => (!item.m_shared.m_useDurability || item.m_durability > 0) &&
                (item.m_shared.m_attack.m_attackType == Attack.AttackType.Horizontal || item.m_shared.m_attack.m_attackType == Attack.AttackType.Vertical));

        // Latch actual damage so food regeneration cannot conceal a hit by
        // restoring health above the value observed at task start.
        internal void Damaged(float amount)
        {
            if (amount <= 0) return;
            injured = true;
            Driver.HarvestAttack = false;
        }

        public Outcome Tick(double dt)
        {
            Driver.Stand();
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (Time.unscaledTime - startedAt > 180) return Finish(Outcome.Failed(Wire.ReasonTimeout, "harvest reached its 180 s work limit"));
            if (injured || HarvestHealth.Hurt(initialHealth, p.GetHealth(), p.GetMaxHealth())) return Finish(Outcome.Failed(Wire.ReasonStopped, "took damage; harvesting cancelled"));
            if (Time.unscaledTime >= nextThreatCheck)
            {
                nextThreatCheck = Time.unscaledTime + .25f;
                if (ThreatNear(p)) return Finish(Outcome.Failed(Wire.ReasonRefused, "an enemy approached within 15 m; harvesting cancelled"));
            }
            if (Walker.Stranded(p) is Outcome stranded) return Finish(stranded);
            if (broken) return AfterBreak(p, dt);
            if (target == null || view == null || !view.IsValid() || !target.activeInHierarchy || !targetColliders.Any(Alive))
            {
                if (!evidence.Broken(Time.unscaledTime)) return Finish(Outcome.Failed(Wire.ReasonTargetLost, "resource disappeared without a recent observed harvest hit"));
                broken = true; brokeAt = Time.unscaledTime;
                Tracer.Emit("harvest", "resource broke after local hit", new Dictionary<string, object>
                {
                    ["entity"] = args.Entity,
                    ["hits"] = evidence.LocalHits,
                });
                return AfterBreak(p, dt);
            }
            origin = target.transform.position;
            if (!PrivateArea.CheckAccess(origin, 0f, false, false)) return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "the resource is behind a ward"));
            if (!Settled(origin, rolling ? 6 : 2)) { swingSince = -1; return null; }
            tool = Best(p);
            if (tool == null) return Finish(Outcome.Failed(Wire.ReasonNoTool, $"need a usable {(rock ? "pickaxe" : "axe")} of tier {minimum}; carried tools may be broken"));
            if (p.GetCurrentWeapon() != tool)
            {
                if (!p.EquipItem(tool, true)) return Finish(Outcome.Failed(Wire.ReasonNoTool, "the selected tool could not be equipped"));
                return null;
            }
            var attackHeight = tool.m_shared.m_attack.m_attackHeight;
            var eye = p.transform.position + Vector3.up * attackHeight;
            if (!HarvestApproach.Contact(targetColliders, origin, p.transform.position, attackHeight, out var surface))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no physical contact on the resource"));
            var reach = (float)HarvestReach.Range(tool.m_shared.m_attack.m_attackRange);
            if (forceApproach && (p.transform.position - walkGoal).sqrMagnitude <= .0225f) forceApproach = false;
            if (forceApproach || !HarvestReach.InRange(Convert.ToV3(p.transform.position), Convert.ToV3(surface), attackHeight, reach) || !UseReach.Near(p, target, 2.5f))
            {
                if (!HarvestApproach.GroundGoal(target, origin, surface, p.transform.position,
                    (float)HarvestReach.StandOff(reach, contactRetries > 0), out var goal))
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no supported, water-safe position beside the resource"));
                if (!walking || Vector3.Distance(walkGoal, goal) > .45f)
                {
                    walker.SetGoal(p.transform.position, goal, .15f);
                    walker.ResetStuck(p.transform.position);
                    walkGoal = goal; walking = true; retreatReady = false;
                    Tracer.Emit("harvest", "approaching physical contact", new Dictionary<string, object>
                    {
                        ["contact"] = Convert.ToArray(surface),
                        ["goal"] = Convert.ToArray(goal),
                        ["reach"] = reach,
                        ["attackHeight"] = attackHeight,
                        ["attackRange"] = tool.m_shared.m_attack.m_attackRange,
                    });
                }
                if (walker.Unreachable(p.transform.position, .15f) is Outcome no) return Finish(no);
                if (walker.Refused() is Outcome refused) return Finish(refused);
                if (!walker.Steer(p.transform.position, false, climbObstacles: Vector3.Distance(p.transform.position, origin) > 3))
                    return Finish(Outcome.Failed(Wire.ReasonStuck, "cannot approach the resource"));
                swingSince = -1;
                return null;
            }
            walking = false;
            if (Protected(p, tallTree ? Mathf.Clamp(targetColliders.Where(Alive).Max(c => c.bounds.max.y) - origin.y + 2, 8, 32) : 0))
                return Finish(Outcome.Failed(Wire.ReasonRefused, "a character or player structure is too close to harvest safely"));
            if (rolling && !retreatReady)
            {
                var away = p.transform.position - origin; away.y = 0;
                if (away.sqrMagnitude < .01f) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no clear side to retreat from the trunk"));
                retreat = origin + away.normalized * 6;
                retreat.y = p.transform.position.y;
                var probe = new WorldNavigation.PlanningProbe(); probe.BeginSlice();
                var at = Convert.ToV3(p.transform.position);
                var end = Convert.ToV3(retreat);
                var count = (int)Math.Ceiling((end - at).Length / .5);
                var start = at;
                for (var i = 1; i <= count; i++)
                {
                    var supported = probe.Probe(at, start + (end - start) * (i / (double)count));
                    if (!(supported is V3 next)) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no supported escape from the falling trunk"));
                    at = next;
                }
                retreat = Convert.ToVector3(at); retreatReady = true;
            }
            Driver.LookAt = surface;
            Driver.LookOriginHeight = attackHeight;
            Driver.LookUntil = Time.time + .2f;
            var direction = surface - eye;
            if (!p.IsOnGround() || p.InAttack() || Vector3.Angle(p.m_lookDir, direction) > 10) return null;
            if (!p.HaveStamina(tool.m_shared.m_attack.m_attackStamina + 1)) { swingSince = -1; return null; }
            if (swingSince < 0 || hitsAtSwingStart != evidence.LocalHits)
            {
                swingSince = Time.unscaledTime; hitsAtSwingStart = evidence.LocalHits;
            }
            if (Time.unscaledTime - swingSince > 8)
            {
                if (contactRetries++ > 0) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "swings did not reach or damage the resource after repositioning"));
                // One bounded closer approach, with fresh support/clearance
                // validation. Never keep swinging indefinitely without a hit.
                if (!HarvestApproach.GroundGoal(target, origin, surface, p.transform.position,
                    (float)HarvestReach.StandOff(reach, true), out var closer))
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no closer supported strike position"));
                walker.SetGoal(p.transform.position, closer, .1f);
                walker.ResetStuck(p.transform.position);
                walkGoal = closer; walking = true; forceApproach = true; swingSince = -1;
                return null;
            }
            Driver.HarvestAttack = true;
            return null;
        }

        bool ThreatNear(Player p)
        {
            // Construction colliders can saturate a broad-phase buffer around
            // home. Use the same character registry as perception so dense
            // scenery cannot masquerade as an enemy or hide one.
            nearbyCharacters.Clear();
            Character.GetCharactersInRange(p.transform.position, 15, nearbyCharacters);
            foreach (var c in nearbyCharacters)
                if (c != null && c != p && !(c is Player) && !c.IsDead() && BaseAI.IsEnemy(p, c)) return true;
            return false;
        }

        bool Protected(Player p, float fallRadius)
        {
            var radius = Math.Max(fallRadius, tool.m_shared.m_attack.m_attackRange + 1.5f);
            var center = fallRadius > 0 ? origin : p.transform.position + Vector3.up;
            var count = Physics.OverlapSphereNonAlloc(center, radius, hits, ~0, QueryTriggerInteraction.Ignore);
            if (count == hits.Length) return true;
            for (var i = 0; i < count; i++)
            {
                var c = hits[i];
                var character = c.GetComponentInParent<Character>();
                if (character != null && character != p || c.GetComponentInParent<Piece>() != null) return true;
            }
            return false;
        }

        bool Settled(Vector3 center, float radius)
        {
            var count = Physics.OverlapSphereNonAlloc(center, radius, hits, ~0, QueryTriggerInteraction.Ignore);
            if (count == hits.Length) return false;
            var quiet = true;
            for (var i = 0; i < count; i++)
            {
                var log = hits[i].GetComponentInParent<TreeLog>();
                if (log == null) continue;
                var body = log.GetComponent<Rigidbody>();
                if (!logPoses.TryGetValue(log, out var pose))
                {
                    if (logPoses.Count == 128) return false;
                    pose = new HarvestStillness();
                    logPoses.Add(log, pose);
                }
                // A remotely simulated rigidbody can have zero local velocity
                // while network updates still move it. Watch its actual pose too.
                var moving = body != null && (body.linearVelocity.sqrMagnitude > .09f || body.angularVelocity.sqrMagnitude > .09f);
                if (!pose.Observe(Convert.ToV3(log.transform.position), Convert.ToV3(log.transform.up),
                    Convert.ToV3(log.transform.forward), moving, Time.unscaledTime)) quiet = false;
            }
            return quiet;
        }

        Outcome AfterBreak(Player p, double dt)
        {
            if (rolling && !retreated && V3.FlatDistance(Convert.ToV3(p.transform.position), Convert.ToV3(origin)) < 5.5)
            {
                if (!retreatReady || Time.unscaledTime - brokeAt > 6) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "could not retreat from the broken trunk"));
                var delta = retreat - p.transform.position; delta.y = 0;
                var probe = new WorldNavigation.PlanningProbe(); probe.BeginSlice();
                if (!probe.Probe(Convert.ToV3(p.transform.position), Convert.ToV3(p.transform.position + delta.normalized * .5f)).HasValue)
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "escape from trunk changed"));
                Driver.LookUntil = 0;
                Driver.Move = delta.normalized; Driver.Run = true;
                return null;
            }
            // Retreat is a phase, not a permanent exclusion radius: after
            // logs settle, collection must be able to approach the drops.
            retreated = true;
            if (!Settled(origin, 8)) { quietSince = -1; return null; }
            if (quietSince < 0) quietSince = Time.unscaledTime;
            if (Time.unscaledTime - quietSince < 2) return null;
            if (!args.Collect) return Complete(p);
            if (pickup != null)
            {
                var result = pickup.Tick(dt);
                if (result == null) return null;
                pickup.Cancel(); pickup = null;
                if (result.Status != Wire.StatusSucceeded && result.Reason != Wire.ReasonTargetLost) return Finish(result);
            }
            if (Time.unscaledTime < nextCollect) return null;
            nextCollect = Time.unscaledTime + .2f;
            var nearby = NearbyDrops();
            if (nearby == null) return Finish(Outcome.Failed(Wire.ReasonRefused, "too many nearby drops to identify safely"));
            var drop = nearby.Where(d => !oldDrops.Contains(d.m_nview.GetZDO().m_uid) && yields.Contains(Name(d.m_itemData)))
                .OrderBy(d => (d.transform.position - p.transform.position).sqrMagnitude).FirstOrDefault();
            if (drop == null) return Complete(p);
            if (++collected > 32 || Time.unscaledTime - brokeAt > 25) return Finish(Outcome.Failed(Wire.ReasonTimeout, "bounded drop collection exhausted; remaining drops left in place"));
            if (!p.GetInventory().CanAddItem(drop.m_itemData, drop.m_itemData.m_stack)) return Finish(Outcome.Failed(Wire.ReasonInventoryFull, "not enough room for the resource drops"));
            if (PickupTask.TooHeavy(p, drop.m_itemData)) return Finish(Outcome.Failed(Wire.ReasonTooHeavy, "resource drops exceed carry weight"));
            pickup = new PickupTask(drop);
            return null;
        }

        ItemDrop[] NearbyDrops()
        {
            var count = Physics.OverlapSphereNonAlloc(origin, 4, hits, LayerMask.GetMask("item"), QueryTriggerInteraction.Collide);
            if (count == hits.Length) return null;
            // Materialize before another bounded scan reuses the buffer.
            return hits.Take(count).Select(c => c.GetComponentInParent<ItemDrop>()).Where(d => d != null && d.m_itemData != null &&
                d.m_nview != null && d.m_nview.IsValid() && Vector3.Distance(d.transform.position, origin) <= 4).Distinct().ToArray();
        }

        Outcome Complete(Player p)
        {
            RenkiPlugin.Instance.Send(InventoryReporter.Describe(p.GetInventory()));
            var gains = p.GetInventory().GetAllItems().GroupBy(Name).Where(g => yields.Contains(g.Key))
                .Select(g => (name: g.Key, count: g.Sum(it => it.m_stack) - (before.TryGetValue(g.Key, out var n) ? n : 0)))
                .Where(g => g.count > 0).Select(g => $"{g.count} {g.name}");
            return Finish(Outcome.Succeeded($"broke {prefab}" + (args.Collect ? $"; pack gain: {string.Join(", ", gains)}" : "; collection disabled")));
        }

        Outcome Finish(Outcome result) { Cancel(); return result; }
        public void Cancel()
        {
            cancelled = true; pickup?.Cancel(); pickup = null;
            if (ReferenceEquals(Active, this)) Active = null;
            Driver.Stand(); Driver.LookUntil = 0;
        }

        internal void Hit(Component resource, HitData hit, bool outgoing)
        {
            if (cancelled || broken || resource == null || target == null || resource.gameObject != target || hit == null || hit.GetTotalDamage() <= 0) return;
            var local = hit.GetAttacker() == Player.m_localPlayer && Player.m_localPlayer != null;
            // Local outgoing Damage and its owner-side RPC describe the same
            // hit; count it once. Owner-side competing hits revoke correlation.
            if (outgoing && local || !outgoing && !local) evidence.Hit(local, Time.unscaledTime);
        }
    }

    [HarmonyPatch]
    static class HarvestHitObservation
    {
        static IEnumerable<MethodBase> TargetMethods()
        {
            foreach (var type in new[] { typeof(TreeBase), typeof(TreeLog), typeof(Destructible), typeof(MineRock), typeof(MineRock5) })
            {
                yield return AccessTools.Method(type, "Damage");
                yield return AccessTools.Method(type, type == typeof(MineRock) ? "RPC_Hit" : "RPC_Damage");
            }
        }
        // Observe normal game damage only. Never create/modify HitData or
        // invoke Damage/RPCs ourselves.
        static void Prefix(Component __instance, object[] __args, MethodBase __originalMethod)
        {
            if (HarvestTask.Active == null) return;
            var hit = __args.OfType<HitData>().FirstOrDefault();
            HarvestTask.Active.Hit(__instance, hit, __originalMethod.Name == "Damage");
        }
    }
}
