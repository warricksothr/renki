using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One native group peel, one verified kill, or bounded escape.</summary>
    sealed class PeelTask : IBodyTask
    {
        readonly string id, entity;
        readonly AttackArgs args;
        readonly Character enemy;
        readonly KiteTask surveyor;
        readonly CharacterDeathObservation death;
        readonly PursuitGroup group;
        readonly Vector3 origin;
        readonly float began;
        readonly Walker walker = new Walker();
        readonly WorldNavigation.PlanningProbe probe = new WorldNavigation.PlanningProbe(dry: true);
        readonly List<Vector3> path = new List<Vector3>();
        readonly List<PursuitObservation> observations = new List<PursuitObservation>();
        IsolationEvidence evidence = new IsolationEvidence();
        Vector3 area, escapeEnd, walkGoal;
        bool planned, walking, cancelled, admitted;
        int surveys;
        float nextSample, nextPlan, nextAim, nextGround, escapeAt;
        string mode = "draw", phase = "prepare", reason;
        MeleeTask melee;
        FleeTask escape;

        internal CombatStatus Status
        {
            get
            {
                var status = melee?.Status ?? new CombatStatus { Id = id, Entity = entity, Mode = mode, Reason = reason };
                status.Peel = new PeelStatus
                {
                    Phase = phase,
                    Members = evidence.Members,
                    Pursuers = evidence.Pursuers,
                    Unknown = evidence.Unknown,
                    QuietS = Math.Round(evidence.QuietS, 2),
                    Escape = planned ? Convert.ToArray(escapeEnd) : null,
                };
                return status;
            }
        }

        public PeelTask(string id, AttackArgs args, Character enemy, CharacterDeathObservation death, KiteTask surveyor)
        {
            this.id = id; this.args = args; this.enemy = enemy; this.death = death; this.surveyor = surveyor;
            entity = enemy.m_nview.GetZDO().m_uid.ToString();
            origin = Player.m_localPlayer.transform.position; began = Time.unscaledTime;
            group = new PursuitGroup(entity);
            walker.ResetStuck(origin);
            Sample(Player.m_localPlayer);
        }

        internal static bool Prepared(Player p)
        {
            var weapon = MeleeTask.Best(p);
            var armour = 1d;
            foreach (var item in new[] { p.m_chestItem, p.m_legItem, p.m_helmetItem, p.m_shoulderItem })
                if (item != null) armour = Math.Min(armour, Durability(item));
            return weapon != null && PursuitGroup.Prepared(p.GetHealth(), p.GetMaxHealth(), p.GetStamina(),
                p.GetFoods().Count > 0, p.GetBodyArmor(), Durability(weapon), armour);
        }

        static double Durability(ItemDrop.ItemData item) => !item.m_shared.m_useDurability ? 1 :
            item.GetMaxDurability() > 0 ? item.m_durability / item.GetMaxDurability() : 0;

        bool Ready(Player p)
        {
            // Drawing may spend stamina, but never its defence/escape reserve.
            var actual = p.GetStamina();
            var weapon = MeleeTask.Best(p);
            var armour = 1d;
            foreach (var item in new[] { p.m_chestItem, p.m_legItem, p.m_helmetItem, p.m_shoulderItem })
                if (item != null) armour = Math.Min(armour, Durability(item));
            return actual >= 20 && weapon != null && PursuitGroup.Prepared(p.GetHealth(), p.GetMaxHealth(), Math.Max(40, actual),
                p.GetFoods().Count > 0, p.GetBodyArmor(), Durability(weapon), armour);
        }

        public Outcome Tick(double dt)
        {
            Driver.Stand();
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "peel stopped"));
            if (escape != null) return EscapeTick(p, dt);
            if (death.Observed || enemy != null && enemy.IsDead()) return Finish(Outcome.Succeeded("observed isolated enemy death"));
            if (!admitted)
            {
                if (!Prepared(p)) return Abort(p, "peel preparation is no longer ready after discovery");
                admitted = true;
            }
            if (!Ready(p) || !p.IsOnGround() || p.InWater() || p.IsSwimming() || p.IsEncumbered())
                return Abort(p, "peel food, gear, health, stamina or dry footing no longer ready");
            if (!PursuitGroup.WithinBudget(Time.unscaledTime - began, Vector3.Distance(origin, p.transform.position)))
                return Abort(p, "peel time or travel budget exhausted");
            if (Time.unscaledTime >= nextSample) Sample(p);
            evidence = group.Assess(Convert.ToV3(p.transform.position), Time.unscaledTime);
            if (enemy == null || enemy.m_nview == null || !enemy.m_nview.IsValid() || !enemy.gameObject.activeInHierarchy)
                return Abort(p, "selected enemy unloaded; death was not observed");
            if (!MeleeTask.Eligible(p, enemy)) return Abort(p, "peel target is no longer supported");
            if (!Separated(p, p.transform.position, 6)) return Abort(p, "another enemy or bystander converged on the peel");
            if (!planned)
            {
                if (Time.unscaledTime >= nextPlan)
                {
                    nextPlan = Time.unscaledTime + .5f;
                    if (!Plan(p) && (args.At.HasValue || surveys >= 8)) return Abort(p, "no checked peel fighting ground and escape route");
                }
                return null;
            }
            if (Time.unscaledTime >= nextGround)
            {
                nextGround = Time.unscaledTime + 1;
                if (!surveyor.TryArea(p, area, out _, out _) || !CheckedRoute(p, area, escapeEnd) || !CheckedRoute(p, p.transform.position, escapeEnd))
                    return Abort(p, "peel fighting ground or escape corridor changed");
            }
            if (melee != null)
            {
                if (!evidence.Isolated) return Abort(p, "isolation lost during melee: " + evidence.Reason);
                var ended = melee.Tick(dt);
                return ended == null ? null : Finish(ended);
            }
            var pos = p.transform.position;
            var gap = Vector3.Distance(pos, enemy.transform.position);
            var following = evidence.Pursuers.Contains(entity);
            if (!following && Time.unscaledTime - began >= 12) return Abort(p, "selected enemy did not pursue within draw budget");
            if (following && Vector3.Distance(pos, area) <= 2 && gap <= 6 && evidence.Isolated)
            {
                Driver.ClearCombat();
                try { melee = new MeleeTask(id, new AttackArgs { Entity = entity, Style = args.Style }, enemy.gameObject, death); }
                catch (TaskRefused e) { return Abort(p, e.Message); }
                SetPhase("fight", "one isolated confirmed pursuer");
                return null;
            }
            if (gap < 4 && !evidence.Isolated) return Abort(p, "target reached strike distance before isolation was confirmed");
            var goal = following ? area : enemy.transform.position + (pos - enemy.transform.position).normalized * 8;
            if (following && Vector3.Distance(pos, area) <= 2)
            {
                SetPhase("isolate", evidence.Reason); walking = false;
                return null;
            }
            SetPhase(following ? "lead" : "draw", evidence.Reason);
            if (!KiteTask.Ground(goal, out goal) || Flee.PastThreat(Convert.ToV3(pos), Convert.ToV3(enemy.transform.position), Convert.ToV3(goal)))
                return Abort(p, "peel destination crosses the enemy or lacks dry support");
            if (!walking || Time.unscaledTime >= nextAim && Vector3.Distance(goal, walkGoal) > 1)
            {
                nextAim = Time.unscaledTime + .5f;
                if (!CheckedRoute(p, pos, goal)) return Abort(p, "peel approach is blocked or crosses the group");
                walkGoal = goal; walking = true;
                walker.SetGoal(pos, goal, .7, dryGround: true); walker.ResetStuck(pos);
            }
            if (walker.Refused() is Outcome refusal) return Abort(p, refusal.Detail ?? refusal.Reason);
            if (!walker.Steer(pos, following && gap < 7 && p.GetStamina() >= 40, climbObstacles: false))
                return Abort(p, "peel walking route failed");
            Driver.LookAt = enemy.transform.position + Vector3.up; Driver.LookUntil = Time.time + .2f;
            return null;
        }

        void Sample(Player p)
        {
            nextSample = Time.unscaledTime + .5f;
            observations.Clear();
            foreach (var c in Character.GetAllCharacters())
            {
                if (c == null || c == p || c is Player || c.m_nview == null || !c.m_nview.IsValid() || c.m_nview.GetZDO() == null) continue;
                var key = c.m_nview.GetZDO().m_uid.ToString();
                if (!group.Contains(key) && (!BaseAI.IsEnemy(p, c) || Vector3.Distance(p.transform.position, c.transform.position) > PursuitGroup.Radius)) continue;
                var ai = c.GetBaseAI();
                var known = ai != null && c.m_nview.IsOwner();
                observations.Add(new PursuitObservation
                {
                    Id = key,
                    Pos = Convert.ToV3(c.transform.position),
                    Loaded = c.gameObject.activeInHierarchy,
                    Dead = c.IsDead(),
                    TargetKnown = known,
                    Targeting = known && ai.GetTargetCreature() == p,
                    HuntingPlayers = known && ai.HuntPlayer(),
                });
            }
            group.Sample(observations, Convert.ToV3(p.transform.position), Time.unscaledTime);
        }

        bool Plan(Player p)
        {
            var away = (origin - enemy.transform.position); away.y = 0;
            if (away.sqrMagnitude < .1f) return false;
            away.Normalize();
            var turn = surveys == 0 ? 0 : (surveys % 2 == 0 ? 1 : -1) * ((surveys + 1) / 2) * 20;
            var candidate = args.At.HasValue ? Convert.ToVector3(args.At.Value) : origin + Quaternion.Euler(0, turn, 0) * away * 20;
            surveys++;
            if (!KiteTask.Ground(candidate, out candidate) || !Separated(p, candidate, PursuitGroup.Separation) ||
                !surveyor.TryArea(p, candidate, out var ground, out _) || !surveyor.EscapeEnd.HasValue) return false;
            if (!KiteTask.Ground(ground + away * 6, out var end)) return false;
            if (!CheckedRoute(p, p.transform.position, ground) || !CheckedRoute(p, ground, end)) return false;
            area = ground; escapeEnd = end; planned = true; nextGround = Time.unscaledTime + 1;
            Tracer.Emit("combat", "peel_ground", new Dictionary<string, object>
            { ["area"] = Convert.ToArray(area), ["escape"] = Convert.ToArray(escapeEnd), ["attempt"] = surveys });
            return true;
        }

        bool CheckedRoute(Player p, Vector3 from, Vector3 to)
        {
            path.Clear(); probe.BeginSlice();
            if (Pathfinding.instance == null || !Pathfinding.instance.GetPath(from, to, path, Pathfinding.AgentType.Humanoid, true, true, false) ||
                path.Count == 0 || Vector3.Distance(path[path.Count - 1], to) > 1.5f) return false;
            var previous = Convert.ToV3(from);
            var corridor = new GroundCorridor.Cursor();
            var length = 0d;
            var selectedGap = enemy == null ? 4 : Math.Min(4, V3.FlatDistance(Convert.ToV3(from), Convert.ToV3(enemy.transform.position)));
            foreach (var corner in path)
            {
                var next = Convert.ToV3(corner); length += (next - previous).Length;
                if (length > 48) return false;
                V3? Inspect(V3 at, V3 desired)
                {
                    var result = probe.Probe(at, desired);
                    if (!result.HasValue || !Separated(p, Convert.ToVector3(result.Value), 6)) return null;
                    // Drawing never deliberately walks into the selected enemy.
                    if (enemy != null && V3.FlatDistance(result.Value, Convert.ToV3(enemy.transform.position)) < selectedGap - .1) return null;
                    return result;
                }
                var count = Math.Max(1, (int)Math.Ceiling((next - previous).Length / 8));
                var segment = previous;
                for (var i = 1; i <= count; i++)
                {
                    var point = previous + (next - previous) * ((double)i / count);
                    if (corridor.Inspect(segment, point, Inspect) != null) return false;
                    segment = point;
                }
                previous = next;
            }
            return true;
        }

        bool Separated(Player p, Vector3 at, double hostileRadius)
        {
            foreach (var c in Character.GetAllCharacters())
                if (c != null && c != p && c != enemy && !c.IsDead() &&
                    Vector3.Distance(at, c.transform.position) < (BaseAI.IsEnemy(p, c) ? hostileRadius : 8)) return false;
            return true;
        }

        Outcome Abort(Player p, string why)
        {
            melee?.Cancel(); melee = null;
            Driver.ClearCombat();
            SetPhase("escape", why); escapeAt = Time.unscaledTime;
            if (!StartEscape(p)) return Finish(Outcome.Failed(Wire.ReasonRefused, why));
            return null;
        }

        bool StartEscape(Player p)
        {
            Character nearest = null; var best = 25f;
            foreach (var c in Character.GetAllCharacters())
                if (c != null && c != p && !c.IsDead() && BaseAI.IsEnemy(p, c) && Vector3.Distance(p.transform.position, c.transform.position) < best)
                { nearest = c; best = Vector3.Distance(p.transform.position, c.transform.position); }
            if (nearest == null) return false;
            var key = nearest.m_nview != null && nearest.m_nview.IsValid() ? nearest.m_nview.GetZDO()?.m_uid.ToString() : null;
            escape = new FleeTask(new FleeArgs { From = key != null ? new Target { Entity = key } : new Target { Pos = Convert.ToV3(nearest.transform.position) }, Distance = 25 },
                nearest.transform.position, nearest.gameObject, planned ? escapeEnd : (Vector3?)null, null, avoidHostiles: true);
            return true;
        }

        Outcome EscapeTick(Player p, double dt)
        {
            if (Time.unscaledTime - escapeAt >= 30) return Finish(Outcome.Failed(Wire.ReasonTimeout, reason + "; peel escape budget exhausted"));
            var result = escape.Tick(dt);
            if (result == null) return null;
            escape.Cancel(); escape = null;
            // Losing the selected threat does not dismiss another pursuer.
            if (result.Status == Wire.StatusSucceeded && StartEscape(p)) return null;
            return Finish(Outcome.Failed(Wire.ReasonRefused, reason + "; escape: " + (result.Detail ?? result.Reason)));
        }

        void SetPhase(string value, string why)
        {
            if (phase == value && reason == why) return;
            phase = value; reason = why;
            mode = value == "escape" ? "retreat" : value == "lead" ? "kite" : value == "isolate" ? "hold" : "draw";
            Tracer.Emit("combat", "peel_" + value, new Dictionary<string, object>
            { ["entity"] = entity, ["reason"] = why, ["members"] = evidence.Members, ["pursuers"] = evidence.Pursuers, ["unknown"] = evidence.Unknown, ["quietS"] = evidence.QuietS });
        }

        Outcome Finish(Outcome result) { Cancel(); return result; }
        public void Cancel()
        {
            if (cancelled) return;
            cancelled = true; melee?.Cancel(); escape?.Cancel(); death.Dispose(); Driver.ClearCombat();
        }
    }
}
