using System;
using System.Collections.Generic;
using System.Linq;
using System.Reflection;
using HarmonyLib;
using Renki.Movement;
using Renki.Perception;
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
        readonly ScanBuffer<Collider> safetyScan = new ScanBuffer<Collider>(256, 2048);
        string scanFailure;
        readonly Dictionary<TreeLog, HarvestStillness> logPoses = new Dictionary<TreeLog, HarvestStillness>();
        readonly HarvestEvidence evidence = new HarvestEvidence();
        readonly Walker walker = new Walker();
        readonly int minimum;
        readonly bool rock, tallTree, rolling;
        readonly string prefab;
        readonly float startedAt = Time.unscaledTime;
        float previousHealth, previousMaximum;
        readonly Dictionary<string, float> foodTimes = new Dictionary<string, float>();
        float nextThreatCheck;
        ItemDrop.ItemData tool;
        Vector3 origin, walkGoal, retreat;
        bool walking, retreatReady, retreated, broken, cancelled, injured, forceApproach, strikeLogged;
        float brokeAt, quietSince = -1, nextCollect, swingSince = -1;
        int hitsAtSwingStart, collected, contactRetries, retreatAttempts;
        int alternativeAttempts;
        bool findingApproach, alternativeApproach;
        Vector3 alternativeAnchor;
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
            previousHealth = p.GetHealth(); previousMaximum = p.GetMaxHealth();
            RefreshedFood(p);
            if (!HarvestHealth.Ready(previousHealth, previousMaximum))
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

        bool RefreshedFood(Player p)
        {
            var foods = p.GetFoods();
            var refreshed = false;
            foreach (var food in foods)
                if (!foodTimes.TryGetValue(food.m_name, out var remaining) || food.m_time > remaining + 1) refreshed = true;
            foodTimes.Clear();
            foreach (var food in foods) foodTimes[food.m_name] = food.m_time;
            return refreshed;
        }

        public Outcome Tick(double dt)
        {
            Driver.Stand();
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (Time.unscaledTime - startedAt > 180) return Finish(Outcome.Failed(Wire.ReasonTimeout, "harvest reached its 180 s work limit"));
            var health = p.GetHealth(); var maximum = p.GetMaxHealth();
            if (HarvestHealth.Interrupted(injured, previousHealth, previousMaximum, health, maximum, RefreshedFood(p)))
                return Finish(Outcome.Failed(Wire.ReasonStopped, "took damage; harvesting cancelled"));
            previousHealth = health; previousMaximum = maximum;
            if (!HarvestHealth.Ready(health, maximum)) return Finish(Outcome.Failed(Wire.ReasonRefused, "health below the harvesting floor"));
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
            if (!Settled(origin, rolling ? 6 : 2))
            {
                swingSince = -1;
                return scanFailure != null ? Finish(Outcome.Failed(Wire.ReasonRefused, scanFailure)) : null;
            }
            tool = Best(p);
            if (tool == null) return Finish(Outcome.Failed(Wire.ReasonNoTool, $"need a usable {(rock ? "pickaxe" : "axe")} of tier {minimum}; carried tools may be broken"));
            if (p.GetCurrentWeapon() != tool)
            {
                if (!p.EquipItem(tool, true)) return Finish(Outcome.Failed(Wire.ReasonNoTool, "the selected tool could not be equipped"));
                return null;
            }
            var attackHeight = tool.m_shared.m_attack.m_attackHeight;
            var reach = (float)HarvestReach.Range(tool.m_shared.m_attack.m_attackRange);
            if (findingApproach)
            {
                if (alternativeAttempts >= HarvestContactSearch.Limit)
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no supported visible log contact after bounded alternative approaches"));
                if (!HarvestApproach.Alternative(p, target, targetColliders, origin, attackHeight, reach, alternativeAttempts++, out var alternative)) return null;
                alternativeAnchor = alternative;
                alternativeApproach = true; findingApproach = false; walking = false; forceApproach = false;
                Tracer.Emit("harvest", "alternative log contact selected", new Dictionary<string, object>
                {
                    ["attempt"] = alternativeAttempts,
                    ["goal"] = Convert.ToArray(alternative),
                    ["resource"] = Convert.ToArray(origin),
                });
            }
            var eye = p.transform.position + Vector3.up * attackHeight;
            var contactFrom = alternativeApproach ? alternativeAnchor : p.transform.position;
            var contactBaseHeight = rolling && !tallTree ? contactFrom.y : origin.y;
            if (!HarvestApproach.Contact(targetColliders, origin, contactBaseHeight, contactFrom, attackHeight, rolling && !tallTree, out var surface, out var contactMethod))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no physical contact on the resource"));
            if (forceApproach && (p.transform.position - walkGoal).sqrMagnitude <= .0225f) forceApproach = false;
            var inRange = HarvestReach.InRange(Convert.ToV3(p.transform.position), Convert.ToV3(surface), attackHeight, reach);
            if (inRange && !forceApproach && !HarvestApproach.ClearStrike(p, target, surface, attackHeight, reach))
                return AnotherApproach("the measured strike contact is occluded or unavailable");
            if (forceApproach || !inRange)
            {
                if (!HarvestApproach.GroundGoal(target, origin, surface, contactFrom,
                    (float)HarvestReach.StandOff(reach, contactRetries > 0), out var goal))
                {
                    Tracer.Emit("harvest", "unsupported physical contact", new Dictionary<string, object>
                    {
                        ["entity"] = args.Entity,
                        ["resource"] = Convert.ToArray(origin),
                        ["contact"] = Convert.ToArray(surface),
                        ["feet"] = Convert.ToArray(p.transform.position),
                        ["probeTop"] = HarvestReach.GroundProbeTop(origin.y, surface.y),
                        ["probe"] = contactMethod,
                        ["colliders"] = targetColliders.Where(Alive).Take(3).Select(c => new
                        {
                            kind = c.GetType().Name,
                            min = Convert.ToArray(c.bounds.min),
                            max = Convert.ToArray(c.bounds.max),
                        }).ToArray(),
                    });
                    return AnotherApproach("no supported, water-safe position beside the resource");
                }
                if (!HarvestReach.InRange(Convert.ToV3(goal), Convert.ToV3(surface), attackHeight, reach))
                {
                    Tracer.Emit("harvest", "contact outside supported strike reach", new Dictionary<string, object>
                    {
                        ["resource"] = Convert.ToArray(origin),
                        ["contact"] = Convert.ToArray(surface),
                        ["goal"] = Convert.ToArray(goal),
                        ["reach"] = reach,
                        ["attackHeight"] = attackHeight,
                        ["probe"] = contactMethod,
                    });
                    return AnotherApproach("the physical contact cannot be reached from supported ground");
                }
                if (!walking || Vector3.Distance(walkGoal, goal) > .45f)
                {
                    walker.SetGoal(p.transform.position, goal, .15f, dryGround: rolling && !tallTree);
                    walker.ResetStuck(p.transform.position);
                    walkGoal = goal; walking = true; retreatReady = false; retreatAttempts = 0;
                    Tracer.Emit("harvest", "approaching physical contact", new Dictionary<string, object>
                    {
                        ["contact"] = Convert.ToArray(surface),
                        ["goal"] = Convert.ToArray(goal),
                        ["reach"] = reach,
                        ["attackHeight"] = attackHeight,
                        ["attackRange"] = tool.m_shared.m_attack.m_attackRange,
                        ["resource"] = Convert.ToArray(origin),
                        ["feet"] = Convert.ToArray(p.transform.position),
                        ["probe"] = contactMethod,
                    });
                }
                if (walker.Unreachable(p.transform.position, .15f) is Outcome no) return AnotherApproach(no.Detail);
                if (walker.Refused() is Outcome refused) return AnotherApproach(refused.Detail);
                if (!walker.Steer(p.transform.position, false, climbObstacles: Vector3.Distance(p.transform.position, origin) > 3))
                    return AnotherApproach("cannot approach the resource");
                swingSince = -1;
                return null;
            }
            walking = false;
            if (!strikeLogged)
            {
                strikeLogged = true;
                Tracer.Emit("harvest", "physical strike contact visible", new Dictionary<string, object>
                {
                    ["entity"] = args.Entity,
                    ["contact"] = Convert.ToArray(surface),
                    ["origin"] = Convert.ToArray(eye),
                    ["reach"] = reach,
                    ["probe"] = contactMethod,
                });
            }
            if (Protected(p, tallTree ? Mathf.Clamp(targetColliders.Where(Alive).Max(c => c.bounds.max.y) - origin.y + 2, 8, 32) : 0, out var protectedReason))
                return Finish(Outcome.Failed(Wire.ReasonRefused, protectedReason));
            if (rolling && !retreatReady)
            {
                var away = p.transform.position - origin; away.y = 0;
                if (away.sqrMagnitude < .01f) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no clear side to retreat from the trunk"));
                if (retreatAttempts == HarvestEscape.CandidateLimit)
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no supported escape from the falling trunk after checking outward alternatives"));
                retreat = Convert.ToVector3(HarvestEscape.Candidate(Convert.ToV3(origin), Convert.ToV3(p.transform.position), retreatAttempts++));
                var probe = new WorldNavigation.PlanningProbe(diagnostics: true); probe.BeginSlice();
                var at = Convert.ToV3(p.transform.position);
                var end = Convert.ToV3(retreat);
                var count = (int)Math.Ceiling((end - at).Length / .5);
                var start = at;
                for (var i = 1; i <= count; i++)
                {
                    var supported = probe.Probe(at, start + (end - start) * (i / (double)count));
                    if (!(supported is V3 next))
                    {
                        Tracer.Emit("harvest", "escape route rejected", new Dictionary<string, object>
                        {
                            ["attempt"] = retreatAttempts,
                            ["goal"] = Convert.ToArray(retreat),
                            ["stage"] = probe.LastFailure?.Stage,
                            ["collider"] = probe.LastFailure?.Collider,
                        });
                        return null;
                    }
                    at = next;
                }
                retreat = Convert.ToVector3(at); retreatReady = true;
                Tracer.Emit("harvest", "escape route selected", new Dictionary<string, object>
                {
                    ["attempt"] = retreatAttempts,
                    ["from"] = Convert.ToArray(p.transform.position),
                    ["goal"] = Convert.ToArray(retreat),
                });
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

        Outcome AnotherApproach(string reason)
        {
            if (!rolling || tallTree) return Finish(Outcome.Failed(Wire.ReasonUnreachable, reason));
            findingApproach = true; alternativeApproach = false; walking = false; swingSince = -1;
            Tracer.Emit("harvest", "searching alternative log contact", new Dictionary<string, object>
            {
                ["reason"] = reason,
                ["attempts"] = alternativeAttempts,
            });
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

        bool ScanSafety(Vector3 center, float radius, string phase)
        {
            var before = safetyScan.Items.Length;
            var complete = safetyScan.TryFill(buffer => Physics.OverlapSphereNonAlloc(center, radius, buffer, ~0, QueryTriggerInteraction.Ignore));
            scanFailure = complete ? null : "too much nearby geometry to check " + phase + " safely";
            if (before != safetyScan.Items.Length || !complete)
                Tracer.Emit("harvest", "safety scan", new Dictionary<string, object>
                {
                    ["phase"] = phase,
                    ["radius"] = radius,
                    ["previousCapacity"] = before,
                    ["capacity"] = safetyScan.Items.Length,
                    ["count"] = safetyScan.Count,
                    ["complete"] = complete,
                });
            return complete;
        }

        bool Protected(Player p, float fallRadius, out string why)
        {
            var radius = Math.Max(fallRadius, tool.m_shared.m_attack.m_attackRange + 1.5f);
            var center = fallRadius > 0 ? origin : p.transform.position + Vector3.up;
            why = null;
            if (!ScanSafety(center, radius, "protection")) { why = scanFailure; return true; }
            for (var i = 0; i < safetyScan.Count; i++)
            {
                var c = safetyScan.Items[i];
                var character = c.GetComponentInParent<Character>();
                if (character is Player peer && peer != p &&
                    !HarvestSafety.PlayerBlocks(rolling || rock || tool.m_shared.m_tamedOnly, Perceiver.Pvp(p), Perceiver.Pvp(peer))) continue;
                var piece = c.GetComponentInParent<Piece>();
                if (character == null || character == p)
                {
                    if (piece == null) continue;
                }
                why = character != null && character != p ? "a character is inside the harvest exclusion" : "a player structure is inside the harvest exclusion";
                Tracer.Emit("harvest", "protected collider", new Dictionary<string, object>
                {
                    ["collider"] = c.transform.root.name + "/" + c.name,
                    ["radius"] = radius,
                    ["center"] = Convert.ToArray(center),
                    ["boundsDistance"] = Math.Sqrt(c.bounds.SqrDistance(center)),
                    ["character"] = character != null && character != p,
                    ["piece"] = piece != null,
                });
                return true;
            }
            return false;
        }

        bool Settled(Vector3 center, float radius)
        {
            if (!ScanSafety(center, radius, "settling")) return false;
            var quiet = true;
            for (var i = 0; i < safetyScan.Count; i++)
            {
                var log = safetyScan.Items[i].GetComponentInParent<TreeLog>();
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
            if (!Settled(origin, 8))
            {
                quietSince = -1;
                return scanFailure != null ? Finish(Outcome.Failed(Wire.ReasonRefused, scanFailure)) : null;
            }
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
