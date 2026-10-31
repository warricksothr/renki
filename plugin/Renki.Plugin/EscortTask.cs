using System;
using System.Collections.Generic;
using Renki.Protocol;
using Renki.Movement;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One body owner: interruptions never leave an old follower steering.</summary>
    sealed class EscortTask : IBodyTask
    {
        internal static EscortTask Active;
        readonly string id;
        readonly EscortArgs args;
        readonly HashSet<string> protectedNames;
        readonly List<Character> characters = new List<Character>();
        readonly Collider[] nearby = new Collider[256];
        readonly HashSet<ZNetView> visited = new HashSet<ZNetView>();
        readonly Dictionary<string, float> attempted = new Dictionary<string, float>();
        IBodyTask child;
        Character opponent;
        ItemDrop.ItemData collectionItem;
        int collectionAmount;
        string mode = "follow", reason, entity;
        Vector3 leaderLast;
        float stillSince, nextScan, childAt, retryAt, nextCollect;
        bool cancelled;

        internal EscortStatus Status => new EscortStatus { Id = id, Leader = args.Player, Mode = mode, Entity = entity, Reason = reason };

        public EscortTask(string id, EscortArgs args)
        {
            this.id = id; this.args = args;
            protectedNames = new HashSet<string>(args.Protect, StringComparer.OrdinalIgnoreCase) { args.Player };
            stillSince = Time.unscaledTime;
            Active = this;
        }

        bool Protected(Player player) => player != null && protectedNames.Contains(player.GetPlayerName());
        bool SafeAlly(Player player) => EscortPolicy.SafeAlly(Protected(player), Perceiver.Pvp(Player.m_localPlayer), Perceiver.Pvp(player));

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) { Cancel(); return Outcome.Failed(Wire.ReasonStopped, "not in the world"); }
            var now = Time.unscaledTime;
            var leader = Players.Find(args.Player);
            if (leader != null && leader.IsDead()) leader = null;
            var visible = leader != null;
            if (visible)
            {
                if (Vector3.Distance(leaderLast, leader.transform.position) > .6f)
                { leaderLast = leader.transform.position; stillSince = now; }
            }
            var gap = visible ? Vector3.Distance(p.transform.position, leader.transform.position) : float.MaxValue;
            // Run a bounded observation scan at 4 Hz, but tick the child every physics step.
            if (now >= nextScan)
            {
                nextScan = now + .25f;
                var threats = Threats(p, leader, out var threat, out var confirmed);
                var ready = EscortPolicy.Ready(p.GetHealth(), p.GetMaxHealth(), p.GetStamina(), p.GetBodyArmor(),
                    p.GetFoods().Count > 0, MeleeTask.Best(p) != null);
                // Existing melee owns its ongoing stamina budget, including block/dodge.
                if (mode == "defend" && child is MeleeTask) ready = p.GetHealth() >= 30 && p.GetHealth() >= p.GetMaxHealth() * .8 && p.GetStamina() >= 10;
                var action = EscortPolicy.Decide(visible, gap, now - stillSince >= 2, threats,
                    confirmed && MeleeTask.Eligible(p, threat), ready && now >= retryAt,
                    args.Defend, args.Collect, HasSpace(p) && (mode != "collect" || collectionItem != null && p.GetInventory().CanAddItem(collectionItem, collectionAmount)), mode == "collect", now - childAt);
                if (mode != "escape")
                {
                    if (action == EscortAction.Escape && threat != null)
                        Escape(threat, threats > 1 ? "multiple threats" : "fight readiness or target evidence is insufficient");
                    else if (action == EscortAction.Defend)
                    {
                        if (mode != "defend" || opponent != threat) Defend(threat);
                    }
                    else if (mode == "defend") Reset("follow", "attacker no longer threatens the protected group");
                    else if (action == EscortAction.Wait) Reset("wait", "leader is out of sight");
                    else if (mode == "collect" && action != EscortAction.Collect)
                    { nextCollect = now + 3; Reset("follow", "collection leash, movement, capacity or time limit"); }
                    else if (mode != "collect" && action == EscortAction.Collect && now >= nextCollect)
                        Collect(p, leader);
                    if (mode == "wait" && visible) Reset("follow", "leader is visible again");
                }
            }
            if (child == null && mode == "follow" && visible && now >= retryAt)
                child = new FollowTask(new FollowArgs { Player = args.Player, Distance = args.Distance }, safeTravel: true);
            if (mode == "collect" && now - childAt >= EscortPolicy.DetourSeconds)
            { nextCollect = now + 3; Reset("follow", "collection time limit"); }
            if (child == null) { Driver.Stand(); return null; }
            var ended = child.Tick(dt);
            if (ended == null && mode == "escape" && now - childAt > 60)
                ended = Outcome.Failed(Wire.ReasonTimeout, "escort escape time limit");
            if (ended != null)
            {
                var previous = mode;
                Reset("follow", previous + ": " + (ended.Detail ?? ended.Reason ?? ended.Status));
                if (previous == "defend" || previous == "escape") retryAt = now + 15;
                else if (previous == "follow") retryAt = now + 2;
                nextCollect = now + 3;
            }
            return null;
        }

        int Threats(Player p, Player leader, out Character threat, out bool confirmed)
        {
            characters.Clear(); Character.GetCharactersInRange(p.transform.position, 30, characters);
            threat = null; confirmed = false; var count = 0; var nearest = float.MaxValue;
            foreach (var c in characters)
            {
                if (c == null || c is Player || c.IsDead() || c.IsTamed() || c.m_nview == null || !c.m_nview.IsValid() || !BaseAI.IsEnemy(p, c)) continue;
                var ai = c.GetBaseAI();
                if (ai is AnimalAI && c != MeleeTask.RecentAttacker) continue;
                var target = ai != null ? ai.GetTargetCreature() : null;
                var victimKnown = target == p || target is Player ally && Protected(ally) && leader != null &&
                    Vector3.Distance(ally.transform.position, leader.transform.position) <= 12;
                victimKnown |= c == MeleeTask.RecentAttacker;
                var distance = Vector3.Distance(p.transform.position, c.transform.position);
                // Unknown remote targets may justify escape, never a new fight.
                if (!(victimKnown && distance <= 16) && !(distance <= 6 && ai != null && ai.HaveTarget())) continue;
                count++;
                if (distance < nearest) { nearest = distance; threat = c; confirmed = victimKnown; }
            }
            return count;
        }

        void Defend(Character threat)
        {
            Reset("defend", "confirmed attacker targets a protected player");
            opponent = threat; entity = Perceiver.Id(threat.m_nview);
            try { child = new MeleeTask(id, new AttackArgs { Entity = entity }, threat.gameObject, safeAlly: SafeAlly); }
            catch (TaskRefused no) { retryAt = Time.unscaledTime + 15; Escape(threat, no.Message); }
        }

        void Escape(Character threat, string why)
        {
            Reset("escape", why); opponent = threat; entity = Perceiver.Id(threat.m_nview);
            child = new FleeTask(new FleeArgs { From = new Target { Entity = entity }, Distance = 25 },
                threat.transform.position, threat.gameObject, null, null);
        }

        static bool HasSpace(Player p) => !p.IsEncumbered() && p.GetInventory().GetTotalWeight() < p.GetMaxCarryWeight() * .9f;
        static bool Material(string prefab) => prefab == "Wood" || prefab == "Stone" || prefab == "Flint" ||
            prefab == "Raspberry" || prefab == "Blueberries" || prefab == "Mushroom" || prefab == "Resin" ||
            prefab == "LeatherScraps" || prefab == "DeerHide" || prefab == "RawMeat" || prefab == "NeckTail" || prefab == "GreydwarfEye";

        void Collect(Player p, Player leader)
        {
            nextCollect = Time.unscaledTime + 2;
            if (leader == null || Math.Abs(p.transform.position.y - leader.transform.position.y) > Arrival.Level || !p.IsOnGround() || p.InWater() || p.IsSwimming() || !HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth())) return;
            var count = Physics.OverlapSphereNonAlloc(p.transform.position, 5, nearby, ~0, QueryTriggerInteraction.Ignore);
            if (count == nearby.Length) return;
            visited.Clear();
            for (var i = 0; i < count; i++)
            {
                var view = nearby[i].GetComponentInParent<ZNetView>();
                if (view == null || !view.IsValid() || !visited.Add(view)) continue;
                var key = Perceiver.Id(view);
                if (attempted.TryGetValue(key, out var until) && Time.unscaledTime < until) continue;
                var obj = view.gameObject; var pos = obj.transform.position;
                if (Vector3.Distance(pos, leader.transform.position) > 6 || Math.Abs(pos.y - p.transform.position.y) > 2 ||
                    !PrivateArea.CheckAccess(pos, 0f, false, false)) continue;
                var drop = obj.GetComponent<ItemDrop>();
                var pick = obj.GetComponent<Pickable>();
                ItemDrop.ItemData data = null; var amount = 0; var prefab = "";
                if (drop != null && drop.m_itemData?.m_dropPrefab != null)
                { data = drop.m_itemData; amount = data.m_stack; prefab = data.m_dropPrefab.name; }
                else if (pick != null && !pick.GetPicked() && pick.m_itemPrefab != null && pick.m_extraDrops.IsEmpty() && pick.m_aggravateRange <= 0)
                {
                    data = pick.m_itemPrefab.GetComponent<ItemDrop>()?.m_itemData; prefab = pick.m_itemPrefab.name;
                    amount = pick.m_dontScale ? pick.m_amount : Math.Max(pick.m_minAmountScaled, Game.instance.ScaleDrops(pick.m_itemPrefab, pick.m_amount));
                    if (pick.m_pickRaiseSkill != Skills.SkillType.None) amount += Math.Max(0, pick.m_bonusYieldAmount);
                }
                if (data == null || amount <= 0 || !Material(prefab) || !p.GetInventory().CanAddItem(data, amount) ||
                    p.GetInventory().GetTotalWeight() + data.GetWeight() / Math.Max(1, data.m_stack) * amount > p.GetMaxCarryWeight() * .9f) continue;
                if (attempted.Count >= 256) attempted.Clear();
                attempted[key] = Time.unscaledTime + 60;
                Reset("collect", "nearby " + prefab); entity = key;
                collectionItem = data; collectionAmount = amount;
                child = drop != null ? (IBodyTask)new PickupTask(drop, safeTravel: true) :
                    new InteractTask(new InteractArgs { Entity = key }, pick, safeTravel: true);
                return;
            }
        }

        void Reset(string next, string why)
        {
            if (mode == next && child == null && reason == why) return;
            child?.Cancel(); child = null; opponent = null; entity = null; collectionItem = null; collectionAmount = 0;
            mode = next; reason = why; childAt = Time.unscaledTime;
            Tracer.Emit("escort", next, new Dictionary<string, object> { ["id"] = id, ["leader"] = args.Player, ["reason"] = why });
        }

        public void Cancel()
        {
            cancelled = true; child?.Cancel(); child = null; Driver.Stand(); Driver.ClearCombat(); MeleeTask.ClearAggressor();
            if (ReferenceEquals(Active, this)) Active = null;
        }
    }
}
