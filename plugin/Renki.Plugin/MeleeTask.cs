using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Explicit bounded melee; every input follows normal game rules.</summary>
    sealed class MeleeTask : IBodyTask
    {
        internal static MeleeTask Active;
        readonly Character enemy;
        readonly string entity, id, prefab;
        Vector3 start;
        readonly MeleeEncounter encounter;
        readonly CharacterDeathObservation death;
        readonly Walker walker = new Walker();
        readonly WorldNavigation.PlanningProbe dodgeProbe = new WorldNavigation.PlanningProbe(dry: true);
        readonly List<Character> characters = new List<Character>();
        readonly Collider[] nearby = new Collider[256];
        ItemDrop.ItemData weapon;
        Attack costAttack;
        FleeTask escape;
        Vector3 walkGoal;
        float nextAim, nextDodge, escapingAt;
        bool walking, cancelled, enemyWasAttacking;
        string mode = "approach", reason;
        MeleeFrame lastFrame;

        internal CombatStatus Status => new CombatStatus { Id = id, Entity = entity, Mode = mode, Reason = reason };

        public MeleeTask(string id, AttackArgs args, GameObject obj, CharacterDeathObservation observation = null)
        {
            this.id = id; entity = args.Entity;
            enemy = obj != null ? obj.GetComponent<Character>() : null;
            var p = Player.m_localPlayer;
            if (enemy == null || !Eligible(p, enemy) || enemy.IsDead())
                throw new TaskRefused(Wire.ReasonInvalidTarget, "melee supports one living unstarred hostile Greyling or wild Boar, never players or tamed creatures");
            prefab = enemy.m_nview.GetPrefabName();
            start = p.transform.position;
            var distance = Vector3.Distance(start, enemy.transform.position);
            if (distance > MeleeEncounter.ClosingRange)
                throw new TaskRefused(Wire.ReasonRefused, "target is outside the 30 m closing range");
            if (!HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth()))
                throw new TaskRefused(Wire.ReasonRefused, "recover to at least 80% health before combat");
            weapon = Best(p);
            if (weapon == null) throw new TaskRefused(Wire.ReasonNoTool, "no usable ordinary melee weapon");
            encounter = new MeleeEncounter(Time.unscaledTime, enemy.GetHealth(), distance);
            if (!encounter.Engaged) mode = "closing";
            walker.ResetStuck(start);
            death = observation ?? new CharacterDeathObservation(enemy, entity);
            Active = this;
        }

        internal static bool Eligible(Player p, Character c) => c != null && c.m_nview != null && c.m_nview.IsValid() &&
            MeleeEncounter.Eligible(c.m_nview.GetPrefabName(), c.GetLevel(), c is Player, c.IsTamed(), BaseAI.IsEnemy(p, c));

        static bool Usable(ItemDrop.ItemData item)
        {
            var a = item.m_shared.m_attack;
            return item.IsWeapon() && !item.m_shared.m_tamedOnly && !a.m_isHomeItem &&
                (!item.m_shared.m_useDurability || item.m_durability > 0) && a.m_attackRange > 0 &&
                (a.m_attackType == Attack.AttackType.Horizontal || a.m_attackType == Attack.AttackType.Vertical) &&
                a.m_attackEitr <= 0 && a.m_attackHealth <= 0 && a.m_attackHealthPercentage <= 0 && !a.m_attackKillsSelf && a.m_attackUseAdrenaline <= 0;
        }

        internal static ItemDrop.ItemData Best(Player p)
        {
            ItemDrop.ItemData best = null; float power = 0;
            foreach (var item in p.GetInventory().GetAllItems())
            {
                if (!Usable(item)) continue;
                var d = item.GetDamage();
                var value = d.m_blunt + d.m_slash + d.m_pierce + d.m_fire + d.m_frost + d.m_lightning + d.m_poison + d.m_spirit;
                if (value > power) { best = item; power = value; }
            }
            return best;
        }

        public Outcome Tick(double dt)
        {
            Driver.Stand();
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (escape != null)
            {
                if (Time.unscaledTime - escapingAt > 60) return Finish(Outcome.Failed(Wire.ReasonTimeout, "retreat time limit"));
                var ended = escape.Tick(dt);
                return ended == null ? null : Finish(Outcome.Failed(ended.Status == Wire.StatusSucceeded ? Wire.ReasonRefused : ended.Reason,
                    reason + "; retreat: " + (ended.Detail ?? ended.Reason)));
            }
            if (death.Observed) return Finish(Outcome.Succeeded($"observed {prefab} death"));
            var valid = enemy != null && enemy.m_nview != null && enemy.m_nview.IsValid() && enemy.gameObject.activeInHierarchy;
            if (!valid) return Finish(Outcome.Failed(Wire.ReasonTargetLost, "target unloaded; death was not observed"));
            if (!Eligible(p, enemy)) return Retreat(p, "target is no longer an eligible hostile");
            if (enemy.IsDead()) return Finish(Outcome.Succeeded($"observed {prefab} death"));

            if (!p.GetInventory().ContainsItem(weapon) || !Usable(weapon)) weapon = Best(p);
            if (weapon == null) return Retreat(p, "no usable melee weapon");
            if (costAttack == null || costAttack.m_weapon != weapon)
            {
                costAttack = weapon.m_shared.m_attack.Clone();
                costAttack.m_character = p; costAttack.m_weapon = weapon;
            }
            var attackCost = Mathf.Max(0, costAttack.GetAttackStamina());
            var dodgeCost = Mathf.Max(0, p.GetDodgeStaminaUse());
            var blockCost = Mathf.Max(p.m_blockStaminaDrain, p.m_perfectBlockStaminaDrain) * (1 + p.GetEquipmentBlockStaminaModifier());
            p.GetSEMan().ModifyBlockStaminaUsage(blockCost, ref blockCost, false);
            blockCost = Mathf.Max(0, blockCost);
            var pos = p.transform.position;
            var targetPos = enemy.transform.position;
            var targetDistance = Vector3.Distance(pos, targetPos);
            if (!encounter.Engaged && targetDistance <= MeleeEncounter.EngagementRange) start = pos;
            var eye = pos + Vector3.up * weapon.m_shared.m_attack.m_attackHeight;
            var collider = enemy.GetCollider();
            if (collider == null || !collider.enabled) return Finish(Outcome.Failed(Wire.ReasonTargetLost, "target collider is unavailable"));
            var contact = collider.ClosestPoint(eye);
            var reach = weapon.m_shared.m_attack.m_attackRange * .8f;
            var attackWindow = enemy.InAttack();
            var incoming = attackWindow && Vector3.Distance(pos, targetPos) <= 3;
            var unsafeArea = Unsafe(p, weapon.m_shared.m_attack.m_attackRange + 1.5f);
            var dodgeDirection = (pos - targetPos); dodgeDirection.y = 0; dodgeDirection.Normalize();
            var canDodge = incoming && !enemyWasAttacking && Time.unscaledTime >= nextDodge && SupportedDodge(pos, dodgeDirection);
            var frame = new MeleeFrame
            {
                Valid = true,
                Eligible = true,
                Weapon = true,
                Unsafe = unsafeArea,
                Hp = p.GetHealth(),
                HpMax = p.GetMaxHealth(),
                Stamina = p.GetStamina(),
                AttackCost = attackCost,
                DodgeCost = dodgeCost,
                BlockCost = blockCost,
                Busy = p.InAttack() || p.InDodge() || p.IsStaggering(),
                Incoming = incoming,
                CanDodge = canDodge,
                CanBlock = p.GetCurrentBlocker()?.GetBlockPower(p.GetSkillFactor(Skills.SkillType.Blocking)) > 0,
                InRange = Vector3.Distance(eye, contact) <= reach,
                TargetHp = enemy.GetHealth(),
                DistanceFromStart = Vector3.Distance(start, pos),
                TargetDistance = targetDistance,
            };
            lastFrame = frame;
            var action = encounter.Decide(frame, Time.unscaledTime);
            enemyWasAttacking = attackWindow;
            SetMode(!encounter.Engaged && action != MeleeAction.Retreat ? "closing" : action.ToString().ToLowerInvariant(), encounter.Reason);
            if (action == MeleeAction.Retreat) return Retreat(p, encounter.Reason);
            if (!p.IsItemEquiped(weapon))
            {
                if (!p.EquipItem(weapon, true)) return Retreat(p, "could not equip melee weapon");
                return null;
            }
            Driver.LookAt = contact; Driver.LookOriginHeight = weapon.m_shared.m_attack.m_attackHeight; Driver.LookUntil = Time.time + .2f;
            var facing = contact - eye;
            var aligned = facing.sqrMagnitude > .001f && Vector3.Angle(p.m_lookDir, facing) < 15;
            if (action == MeleeAction.Approach || action == MeleeAction.Closing)
            {
                if (!walking || Time.unscaledTime >= nextAim && Vector3.Distance(targetPos, walkGoal) > .6f)
                {
                    if (!HarvestApproach.GroundGoal(enemy.gameObject, targetPos, contact, pos, Mathf.Min(1, reach * .65f), out var goal))
                        return Retreat(p, "no supported strike stance");
                    walker.SetGoal(pos, goal, .2, dryGround: true); walker.ResetStuck(pos); walkGoal = targetPos; walking = true; nextAim = Time.unscaledTime + .5f;
                }
                if (walker.Refused() is Outcome refusal) return Retreat(p, refusal.Detail ?? refusal.Reason);
                if (!walker.Steer(pos, MeleeEncounter.RunApproach(frame, reach), climbObstacles: false)) return Retreat(p, "cannot approach target");
            }
            else
            {
                walking = false;
                if (action == MeleeAction.Strike && aligned) Driver.MeleeAttack = true;
                if (action == MeleeAction.Block) Driver.Block = true;
                if (action == MeleeAction.Dodge && aligned)
                {
                    Driver.Move = dodgeDirection; Driver.Block = Driver.Dodge = true; nextDodge = Time.unscaledTime + 2;
                }
            }
            return null;
        }

        bool Unsafe(Player p, float radius)
        {
            if (!p.IsOnGround() || p.InWater() || p.IsSwimming() || p.IsEncumbered() || p.InPlaceMode()) return true;
            characters.Clear(); Character.GetCharactersInRange(p.transform.position, Math.Max(8, radius), characters);
            foreach (var c in characters)
            {
                if (c == null || c == p || c == enemy || c.IsDead()) continue;
                var distance = Vector3.Distance(p.transform.position, c.transform.position);
                if (distance <= radius || distance <= 6 && BaseAI.IsEnemy(p, c)) return true;
            }
            var count = Physics.OverlapSphereNonAlloc(p.transform.position + Vector3.up, radius, nearby, ~0, QueryTriggerInteraction.Ignore);
            if (count == nearby.Length) return true;
            for (var i = 0; i < count; i++) if (nearby[i].GetComponentInParent<Piece>() != null) return true;
            return false;
        }

        bool SupportedDodge(Vector3 from, Vector3 direction)
        {
            if (direction.sqrMagnitude < .1f) return false;
            dodgeProbe.BeginSlice();
            V3? Probe(V3 previous, V3 desired)
            {
                var next = dodgeProbe.Probe(previous, desired);
                if (!(next is V3 supported) || ZoneSystem.instance == null || supported.Y <= ZoneSystem.instance.m_waterLevel + .2f) return null;
                foreach (var c in characters)
                    if (c != null && c != Player.m_localPlayer && !c.IsDead() &&
                        V3.FlatDistance(supported, Convert.ToV3(c.transform.position)) < 1.5) return null;
                return supported;
            }
            return GroundCorridor.Check(Convert.ToV3(from), Convert.ToV3(from + direction * 5), Probe, out _);
        }

        Outcome Retreat(Player p, string why)
        {
            death.Dispose();
            Driver.ClearCombat(); SetMode("retreat", why); escapingAt = Time.unscaledTime;
            escape = new FleeTask(new FleeArgs { From = new Target { Entity = entity }, Distance = 25 },
                enemy != null ? enemy.transform.position : start, enemy != null ? enemy.gameObject : null, null, null);
            return null;
        }

        void SetMode(string next, string why)
        {
            if (mode == next && reason == why) return;
            mode = next; reason = why;
            Tracer.Emit("combat", next, new Dictionary<string, object>
            {
                ["entity"] = entity,
                ["reason"] = why,
                ["travelFromStart"] = lastFrame?.DistanceFromStart,
                ["targetDistance"] = lastFrame?.TargetDistance,
                ["targetHealth"] = lastFrame?.TargetHp,
                ["health"] = lastFrame?.Hp,
                ["stamina"] = lastFrame?.Stamina,
            });
        }

        Outcome Finish(Outcome result) { result.Entity = entity; Cancel(); return result; }
        public void Cancel()
        {
            cancelled = true; death?.Dispose(); escape?.Cancel();
            if (ReferenceEquals(Active, this)) Active = null;
            Driver.ClearCombat();
        }
    }
}
