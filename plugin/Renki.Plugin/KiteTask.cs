using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One native goal: survey, acquire, draw, lead, then melee. No controller movement loop.</summary>
    sealed class KiteTask : IBodyTask
    {
        internal static KiteTask Active;
        readonly string id;
        readonly AttackArgs args;
        readonly Vector3 origin;
        readonly float began;
        readonly Walker walker = new Walker();
        readonly WorldNavigation.PlanningProbe probe = new WorldNavigation.PlanningProbe(diagnostics: true);
        readonly List<Character> characters = new List<Character>();
        readonly List<Vector3> approachPath = new List<Vector3>();
        readonly Collider[] nearby = new Collider[256];
        Character enemy;
        string entity = "", mode = "survey", reason;
        Vector3 area, lastEnemy, walkGoal;
        Kiting policy;
        GroundCorridor.Failure groundFailure;
        MeleeTask melee;
        FleeTask escape;
        int survey;
        bool hasArea, positioned, walking, cancelled;
        float nextSurvey, nextAreaCheck, nextSample, followingUntil, nextAim, escapeAt, searchAt, nextSearch;

        internal CombatStatus Status => melee != null ? melee.Status :
            new CombatStatus { Id = id, Entity = entity, Mode = mode, Reason = reason };

        public KiteTask(string id, AttackArgs args)
        {
            this.id = id; this.args = args;
            var p = Player.m_localPlayer;
            origin = p.transform.position; began = Time.unscaledTime;
            if (!HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth()) || p.GetStamina() < 40)
                throw new TaskRefused(Wire.ReasonRefused, "recover to 80% health and 40 stamina before drawing an enemy");
            if (MeleeTask.Best(p) == null) throw new TaskRefused(Wire.ReasonNoTool, "no usable melee weapon");
            if (args.At.HasValue && Vector3.Distance(origin, Convert.ToVector3(args.At.Value)) > 12)
                throw new TaskRefused(Wire.ReasonRefused, "preferred fighting ground must be within 12 m");
            walker.ResetStuck(origin); Active = this;
        }

        public Outcome Tick(double dt)
        {
            Driver.Stand();
            var p = Player.m_localPlayer;
            if (cancelled || p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (melee != null)
            {
                var ended = melee.Tick(dt);
                return ended == null ? null : Finish(ended);
            }
            if (escape != null)
            {
                if (Time.unscaledTime - escapeAt >= 30) return Finish(Outcome.Failed(Wire.ReasonTimeout, reason + "; escape time limit"));
                var ended = escape.Tick(dt);
                return ended == null ? null : Finish(Outcome.Failed(Wire.ReasonRefused, reason + "; retreat: " + (ended.Detail ?? ended.Reason)));
            }
            if (!HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth()) || p.GetStamina() < 10 || MeleeTask.Best(p) == null)
                return Abort(p, "health, stamina or equipment no longer supports kiting");
            if (!p.IsOnGround() || p.IsSwimming() || p.IsEncumbered()) return Abort(p, "unsafe footing");
            if (policy == null && Time.unscaledTime - began >= 25)
                return Finish(Outcome.Failed(Wire.ReasonTimeout, "local encounter preparation time limit"));
            if (!hasArea)
            {
                if (Time.unscaledTime < nextSurvey) return null;
                nextSurvey = Time.unscaledTime + .25f;
                Vector3 candidate;
                if (args.At.HasValue) candidate = Convert.ToVector3(args.At.Value);
                else
                {
                    candidate = Convert.ToVector3(Kiting.SurveyPoint(Convert.ToV3(origin), survey));
                    if (Ground(candidate, out var projected)) candidate = projected;
                }
                survey++;
                if (TryArea(p, candidate, out area, out var rejected))
                {
                    hasArea = true; nextAreaCheck = Time.unscaledTime + 1;
                    Mode("position");
                }
                else
                {
                    TraceGroundRejection(candidate, rejected);
                    if (args.At.HasValue ? survey >= 4 : survey >= Kiting.SurveyAttempts)
                        return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no supported clear fighting ground with a checked escape corridor; last rejection: " + rejected));
                }
                return null;
            }
            if (Time.unscaledTime >= nextAreaCheck)
            {
                nextAreaCheck = Time.unscaledTime + 1;
                if (!TryArea(p, area, out _, out var rejected)) return Abort(p, "fighting ground changed or became unsafe: " + rejected);
            }
            if (enemy == null && policy == null)
            {
                if (!positioned)
                {
                    var stagingPosition = p.transform.position;
                    if (Vector3.Distance(stagingPosition, area) > 2)
                    {
                        if (!walking) { walker.SetGoal(stagingPosition, area, 1); walker.ResetStuck(stagingPosition); walking = true; }
                        if (walker.Refused() is Outcome no) return Abort(p, no.Detail ?? no.Reason);
                        if (!walker.Steer(stagingPosition, false, climbObstacles: false)) return Abort(p, "cannot reach fighting ground");
                        return null;
                    }
                    positioned = true; walking = false; searchAt = Time.unscaledTime; Mode("search");
                }
                if (Time.unscaledTime < nextSearch) return null;
                nextSearch = Time.unscaledTime + .5f;
                Acquire(p);
                if (enemy == null)
                {
                    if (Time.unscaledTime - searchAt >= 8)
                        return Finish(Outcome.Failed(Wire.ReasonInvalidTarget, "no eligible single Greyling in the local search area"));
                    return null;
                }
                entity = enemy.m_nview.GetZDO().m_uid.ToString();
                lastEnemy = enemy.transform.position; nextSample = Time.unscaledTime + .5f;
                policy = new Kiting(Convert.ToV3(origin), Convert.ToV3(area), Convert.ToV3(lastEnemy), Time.unscaledTime);
                Mode("draw");
            }
            if (enemy == null || enemy.m_nview == null || !enemy.m_nview.IsValid() || !enemy.gameObject.activeInHierarchy)
                return Finish(Outcome.Failed(Wire.ReasonTargetLost, "target unloaded; death was not observed"));
            if (enemy.IsDead()) return Finish(Outcome.Succeeded("observed Greyling death"));
            var pos = p.transform.position; var target = enemy.transform.position;
            if (Time.unscaledTime >= nextSample)
            {
                nextSample = Time.unscaledTime + .5f;
                if (Kiting.ObservedPursuit(Convert.ToV3(lastEnemy), Convert.ToV3(target), Convert.ToV3(pos)))
                    followingUntil = Time.unscaledTime + 1;
                lastEnemy = target;
            }
            var frame = new KiteFrame
            {
                Bot = Convert.ToV3(pos),
                Enemy = Convert.ToV3(target),
                Hp = p.GetHealth(),
                HpMax = p.GetMaxHealth(),
                Stamina = p.GetStamina(),
                Eligible = MeleeTask.Eligible(p, enemy),
                Safe = ClearCharacters(p, enemy, pos) && ClearCharacters(p, enemy, target) && ClearScenery(pos),
                Following = Time.unscaledTime < followingUntil,
            };
            var action = policy.Decide(frame, Time.unscaledTime);
            if (action == KiteAction.Retreat) return Abort(p, policy.Reason);
            if (action == KiteAction.Engage)
            {
                if (!TryArea(p, area, out _, out var rejected)) return Abort(p, "fighting ground became unsafe before engagement: " + rejected);
                Driver.ClearCombat();
                try { melee = new MeleeTask(id, new AttackArgs { Entity = entity, Style = args.Style }, enemy.gameObject); }
                catch (TaskRefused refused) { return Abort(p, refused.Message); }
                Mode("engage");
                return null;
            }
            Mode(action == KiteAction.Lead ? "kite" : action.ToString().ToLowerInvariant());
            Driver.LookAt = target + Vector3.up; Driver.LookUntil = Time.time + .2f;
            if (action == KiteAction.Hold) { walking = false; return null; }
            if (action == KiteAction.Lead && Vector3.Distance(pos, area) <= 2) { walking = false; Mode("hold"); return null; }
            var goal = Convert.ToVector3(policy.Goal(frame, action));
            if (!Ground(goal, out goal)) return Abort(p, "no supported kiting destination");
            if (!walking || Time.unscaledTime >= nextAim && Vector3.Distance(walkGoal, goal) > 1)
            {
                walkGoal = goal; nextAim = Time.unscaledTime + .5f; walking = true;
                walker.SetGoal(pos, goal, .7); walker.ResetStuck(pos);
            }
            if (walker.Refused() is Outcome refusal) return Abort(p, refusal.Detail ?? refusal.Reason);
            if (!walker.Steer(pos, policy.Run, climbObstacles: false)) return Abort(p, "cannot follow the kiting route");
            if (action == KiteAction.Lead && Vector3.Distance(pos, target) < 5 &&
                Vector3.Dot(Driver.Move, (target - pos).normalized) > .5f)
                return Abort(p, "kiting path would cross the enemy at close range");
            return null;
        }

        void Acquire(Player p)
        {
            characters.Clear(); Character.GetCharactersInRange(p.transform.position, (float)Kiting.SearchRange, characters);
            Character selected = null; var best = float.PositiveInfinity;
            foreach (var c in characters)
            {
                if (!MeleeTask.Eligible(p, c) || c.IsDead() || c.m_nview.GetZDO() == null) continue;
                if (!string.IsNullOrWhiteSpace(args.Entity) && c.m_nview.GetZDO().m_uid.ToString() != args.Entity) continue;
                if (Vector3.Distance(origin, c.transform.position) > Kiting.SearchRange) continue;
                if (!ClearCharacters(p, c, c.transform.position) || !ClearCharacters(p, c, p.transform.position)) continue;
                var distance = Vector3.Distance(p.transform.position, c.transform.position);
                if (distance < best) { selected = c; best = distance; }
            }
            enemy = selected;
        }

        bool ClearCharacters(Player p, Character allowed, Vector3 at)
        {
            // Separate buffer: acquisition iterates characters while this runs.
            foreach (var c in Character.GetAllCharacters())
            {
                if (c == null || c == p || c == allowed || c.IsDead()) continue;
                var radius = BaseAI.IsEnemy(p, c) ? 15 : 8;
                if (Vector3.Distance(at, c.transform.position) < radius) return false;
            }
            return true;
        }

        static bool Ground(Vector3 at, out Vector3 ground)
        {
            // GetWorldHeight rounds to a mesh vertex. Use the game's terrain-only
            // ray projection so step/grade probes start on the actual surface.
            ground = at;
            if (ZoneSystem.instance == null || Heightmap.FindHeightmap(at) == null ||
                !ZoneSystem.instance.GetGroundHeight(at, out var height)) return false;
            ground.y = height;
            return height > ZoneSystem.instance.m_waterLevel + .2f;
        }

        bool ClearScenery(Vector3 at)
        {
            var count = Physics.OverlapSphereNonAlloc(at + Vector3.up, 4, nearby, ~0, QueryTriggerInteraction.Ignore);
            if (count == nearby.Length) return false;
            for (var i = 0; i < count; i++) if (nearby[i].GetComponentInParent<Piece>() != null) return false;
            return true;
        }

        bool TryArea(Player p, Vector3 at, out Vector3 grounded, out string rejected)
        {
            groundFailure = null;
            probe.BeginSlice();
            rejected = "wet_or_unloaded_terrain";
            if (!Ground(at, out grounded)) return false;
            rejected = "different_ground_level";
            if (Math.Abs(at.y - grounded.y) > 1) return false;
            // Full diameters catch a sharp crest/dip at the center as well as
            // validate body-width approaches and six-metre escape corridors.
            var center = Convert.ToV3(grounded);
            V3? DryProbe(V3 from, V3 desired)
            {
                var result = probe.Probe(from, desired);
                if (!result.HasValue) return null;
                return ZoneSystem.instance != null && result.Value.Y > ZoneSystem.instance.m_waterLevel + .2f
                    ? result : probe.Refuse("wet_ground", result.Value);
            }
            for (var i = 0; i < 4; i++)
            {
                var angle = i * Math.PI / 4;
                var radius = new V3(Math.Cos(angle), 0, Math.Sin(angle)) * Kiting.AreaRadius;
                rejected = "wet_or_unloaded_terrain";
                if (!Ground(Convert.ToVector3(center - radius), out var edge)) return false;
                groundFailure = GroundCorridor.Inspect(Convert.ToV3(edge), center + radius, DryProbe);
                if (groundFailure != null) { rejected = groundFailure.Reason; return false; }
            }
            var count = Physics.OverlapSphereNonAlloc(grounded + Vector3.up, 8, nearby, ~0, QueryTriggerInteraction.Ignore);
            rejected = "crowded_geometry";
            if (count == nearby.Length) return false;
            for (var i = 0; i < count; i++)
            {
                rejected = "structure";
                if (nearby[i].GetComponentInParent<Piece>() != null) return false;
                var c = nearby[i].GetComponentInParent<Character>();
                rejected = "bystander_or_other_creature";
                if (c != null && c != p && c != enemy && !c.IsDead() && !MeleeTask.Eligible(p, c)) return false;
            }
            // Validate a usable approach before drawing, not just an empty destination.
            if (Vector3.Distance(p.transform.position, grounded) > 2)
            {
                approachPath.Clear();
                rejected = "no_navmesh_approach";
                if (Pathfinding.instance == null || !Pathfinding.instance.GetPath(p.transform.position, grounded, approachPath,
                    Pathfinding.AgentType.Humanoid, true, true, false) || approachPath.Count == 0 || Vector3.Distance(approachPath[approachPath.Count - 1], grounded) > 1.5f)
                    return false;
            }
            rejected = null;
            return true;
        }

        void TraceGroundRejection(Vector3 candidate, string rejected)
        {
            var data = new Dictionary<string, object>
            { ["candidate"] = Convert.ToArray(candidate), ["reason"] = rejected, ["attempt"] = survey };
            if (groundFailure != null)
            {
                data["segment"] = groundFailure.Segment;
                data["from"] = groundFailure.From.ToArray(); data["desired"] = groundFailure.Desired.ToArray();
                if (groundFailure.Supported.HasValue) data["supported"] = groundFailure.Supported.Value.ToArray();
                if (groundFailure.Angle.HasValue) data["angle"] = Math.Round(groundFailure.Angle.Value, 2);
                if (groundFailure.PreviousAngle.HasValue) data["previousAngle"] = Math.Round(groundFailure.PreviousAngle.Value, 2);
                if (groundFailure.Roughness.HasValue) data["roughness"] = Math.Round(groundFailure.Roughness.Value, 3);
            }
            var physical = probe.LastFailure;
            if (physical != null)
            {
                data["probeStage"] = physical.Stage; data["probeAt"] = physical.At.ToArray();
                if (physical.Collider != null) data["collider"] = physical.Collider;
                if (physical.ColliderCenter.HasValue) data["colliderCenter"] = physical.ColliderCenter.Value.ToArray();
            }
            Tracer.Emit("combat", "ground_rejected", data);
        }

        Outcome Abort(Player p, string why)
        {
            Driver.ClearCombat();
            Mode("retreat", why);
            // A distant lost follower is no reason to start another long chase.
            if (enemy == null || Vector3.Distance(p.transform.position, enemy.transform.position) >= 15)
                return Finish(Outcome.Failed(Wire.ReasonRefused, why));
            escapeAt = Time.unscaledTime;
            escape = new FleeTask(new FleeArgs { From = new Target { Entity = entity }, Distance = 25 },
                enemy.transform.position, enemy.gameObject, null, null);
            return null;
        }

        void Mode(string value, string why = null)
        {
            if (mode == value && reason == why) return;
            mode = value; reason = why;
            Tracer.Emit("combat", value, new Dictionary<string, object>
            {
                ["entity"] = entity,
                ["reason"] = reason,
                ["area"] = Convert.ToArray(area),
                ["gap"] = enemy != null ? Vector3.Distance(Player.m_localPlayer.transform.position, enemy.transform.position) : 0,
                ["stamina"] = Player.m_localPlayer.GetStamina(),
            });
        }

        Outcome Finish(Outcome result) { Cancel(); return result; }
        public void Cancel()
        {
            cancelled = true; melee?.Cancel(); escape?.Cancel();
            if (ReferenceEquals(Active, this)) Active = null;
            Driver.ClearCombat();
        }
    }
}
