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
        readonly double searchRadius;
        readonly Walker walker = new Walker();
        readonly WorldNavigation.PlanningProbe probe = new WorldNavigation.PlanningProbe(diagnostics: true);
        readonly List<Character> characters = new List<Character>();
        readonly List<Vector3> approachPath = new List<Vector3>();
        readonly Collider[] nearby = new Collider[256];
        Character enemy;
        string entity = "", mode = "survey", reason;
        Vector3 area, lastEnemy, walkGoal, surveyOrigin, encounterOrigin;
        Kiting policy;
        Scouting scouting;
        ScoutAction scoutAction;
        GroundCorridor.Failure groundFailure;
        string groundRegion;
        V3? escapeEnd;
        MeleeTask melee;
        FleeTask escape;
        MoveToTask preparation;
        Vector3 preparationGoal, preparationLast;
        float preparationBegan, preparationLegAt, preparationTravel;
        int preparationAttempts;
        bool preparing;
        int survey;
        bool hasArea, areaChecked, positioned, walking, cancelled;
        float nextSurvey, nextAreaCheck, nextSample, followingUntil, nextAim, escapeAt, nextSearch, nextScout;
        float surveyBegan;

        internal CombatStatus Status => melee != null ? melee.Status :
            new CombatStatus { Id = id, Entity = entity, Mode = mode, Reason = reason };

        public KiteTask(string id, AttackArgs args)
        {
            this.id = id; this.args = args;
            var p = Player.m_localPlayer;
            origin = p.transform.position; began = Time.unscaledTime;
            surveyOrigin = encounterOrigin = origin; surveyBegan = began;
            searchRadius = args.SearchRadius ?? (string.IsNullOrWhiteSpace(args.Entity) && !args.At.HasValue ? 60 : 12);
            if (!HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth()) || p.GetStamina() < 40)
                throw new TaskRefused(Wire.ReasonRefused, "recover to 80% health and 40 stamina before drawing an enemy");
            if (MeleeTask.Best(p) == null) throw new TaskRefused(Wire.ReasonNoTool, "no usable melee weapon");
            if (args.At.HasValue && Vector3.Distance(origin, Convert.ToVector3(args.At.Value)) > 12)
                throw new TaskRefused(Wire.ReasonRefused, "preferred fighting ground must be within 12 m");
            walker.ResetStuck(origin);
            if (string.IsNullOrWhiteSpace(args.Entity) && !args.At.HasValue)
            {
                // Searching needs a safe walking route, not an arena at the
                // starting spot. Survey combat ground only after discovery.
                if (!Ground(origin, out area))
                    throw new TaskRefused(Wire.ReasonUnreachable, "no dry supported search origin");
                hasArea = positioned = true;
                preparing = Math.Abs(origin.y - area.y) > 1 || !ClearScenery(origin);
                if (preparing)
                {
                    preparationLast = origin; preparationBegan = began;
                    Mode("prepare_search", "travel from building support to open ground before scouting");
                }
                else
                {
                    scouting = new Scouting(Convert.ToV3(origin), Convert.ToV3(area), Convert.ToV3(origin), Time.unscaledTime, searchRadius);
                    Mode("search");
                }
            }
            Active = this;
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
            if (entity != "" && (enemy == null || enemy.m_nview == null || !enemy.m_nview.IsValid() || !enemy.gameObject.activeInHierarchy))
                return Finish(Outcome.Failed(Wire.ReasonTargetLost, "target unloaded; death was not observed"));
            if (!HarvestHealth.Ready(p.GetHealth(), p.GetMaxHealth()) || p.GetStamina() < 10 || MeleeTask.Best(p) == null)
                return Abort(p, "health, stamina or equipment no longer supports kiting");
            if (!p.IsOnGround() || p.IsSwimming() || p.IsEncumbered()) return Abort(p, "unsafe footing");
            if (Vector3.Distance(origin, p.transform.position) > searchRadius + Kiting.TravelLimit)
                return Abort(p, "overall search and encounter travel limit");
            if (preparing) return PrepareSearch(p, dt);
            if (!positioned && Time.unscaledTime - surveyBegan >= 25)
                return Finish(Outcome.Failed(Wire.ReasonTimeout, "local encounter preparation time limit"));
            if (!hasArea)
            {
                if (Time.unscaledTime < nextSurvey) return null;
                nextSurvey = Time.unscaledTime + .25f;
                Vector3 candidate;
                if (args.At.HasValue) candidate = Convert.ToVector3(args.At.Value);
                else
                {
                    candidate = Convert.ToVector3(Kiting.SurveyPoint(Convert.ToV3(surveyOrigin), survey));
                    if (Ground(candidate, out var projected)) candidate = projected;
                }
                survey++;
                if (TryArea(p, candidate, out area, out var rejected))
                {
                    hasArea = areaChecked = true; nextAreaCheck = Time.unscaledTime + 1;
                    Tracer.Emit("combat", "ground_selected", new Dictionary<string, object>
                    {
                        ["area"] = Convert.ToArray(area),
                        ["escape"] = escapeEnd.Value.ToArray(),
                        ["stanceRadius"] = FightingGround.StanceRadius,
                    });
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
            if (Time.unscaledTime >= nextAreaCheck && (enemy != null || !positioned))
            {
                nextAreaCheck = Time.unscaledTime + 1;
                if (!TryArea(p, area, out _, out var rejected)) return Abort(p, "fighting ground changed or became unsafe: " + rejected);
            }
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
                positioned = true; walking = false; Mode("search");
                if (enemy == null)
                    scouting = new Scouting(Convert.ToV3(origin), Convert.ToV3(area), Convert.ToV3(stagingPosition), Time.unscaledTime, searchRadius);
            }
            if (enemy == null && policy == null)
            {
                if (Time.unscaledTime >= nextSearch)
                {
                    nextSearch = Time.unscaledTime + .5f;
                    Acquire(p);
                }
                if (enemy == null)
                    return Scout(p);
                walking = false;
                entity = enemy.m_nview.GetZDO().m_uid.ToString();
                encounterOrigin = p.transform.position;
                if (!areaChecked || Kiting.NeedsLocalArea(Convert.ToV3(encounterOrigin), Convert.ToV3(area), args.At.HasValue))
                {
                    // Search can range widely; the lure stays local to the
                    // discovery rather than dragging enemies back to spawn.
                    surveyOrigin = encounterOrigin; surveyBegan = Time.unscaledTime;
                    hasArea = positioned = false; survey = 0; nextSurvey = 0;
                    Mode("survey", "choose fighting ground near the discovered enemy");
                    return null;
                }
            }
            if (policy == null && enemy != null)
            {
                if (!TryArea(p, area, out _, out var rejected)) return Abort(p, "fighting ground became unsafe before drawing: " + rejected);
                lastEnemy = enemy.transform.position; nextSample = Time.unscaledTime + .5f;
                policy = new Kiting(Convert.ToV3(encounterOrigin), Convert.ToV3(area), Convert.ToV3(lastEnemy), Convert.ToV3(p.transform.position), Time.unscaledTime);
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
                Targeting = enemy.GetBaseAI() != null && enemy.GetBaseAI().GetTargetCreature() == p,
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
                if (Vector3.Distance(origin, c.transform.position) > searchRadius + Kiting.SearchRange) continue;
                if (!ClearCharacters(p, c, c.transform.position) || !ClearCharacters(p, c, p.transform.position)) continue;
                var distance = Vector3.Distance(p.transform.position, c.transform.position);
                if (distance < best) { selected = c; best = distance; }
            }
            enemy = selected;
        }

        Outcome PrepareSearch(Player p, double dt)
        {
            Driver.ClearCombat();
            var pos = p.transform.position;
            preparationTravel += Vector3.Distance(pos, preparationLast); preparationLast = pos;
            if (Time.unscaledTime - preparationBegan >= 60 || preparationTravel > 64 || Vector3.Distance(origin, pos) > searchRadius + .25)
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "search travel preparation exhausted its time or travel budget"));
            if (!ClearCharacters(p, null, pos)) return Abort(p, "another character is too close during search travel preparation");
            if (preparation == null)
            {
                if (preparationAttempts >= 8)
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "no reachable clear search origin after bounded travel preparation"));
                var angle = preparationAttempts++ * Math.PI / 4;
                var candidate = origin + new Vector3((float)Math.Cos(angle), 0, (float)Math.Sin(angle)) * (float)Math.Min(16, searchRadius - 1);
                if (!Ground(candidate, out preparationGoal) || Vector3.Distance(origin, preparationGoal) > searchRadius ||
                    !ClearScenery(preparationGoal) || !ClearCharacters(p, null, preparationGoal))
                {
                    Tracer.Emit("combat", "search_egress_rejected", new Dictionary<string, object>
                    { ["goal"] = Convert.ToArray(candidate), ["reason"] = "endpoint_not_dry_clear_or_separated", ["attempt"] = preparationAttempts });
                    return null;
                }
                preparation = new MoveToTask(new MoveToArgs
                { To = new Target { Pos = Convert.ToV3(preparationGoal) }, StopDistance = 1, Run = false }, preparationGoal, null);
                preparationLegAt = Time.unscaledTime;
                Mode("prepare_search", "ordinary checked travel to open ground");
                Tracer.Emit("combat", "search_egress_route", new Dictionary<string, object>
                { ["goal"] = Convert.ToArray(preparationGoal), ["attempt"] = preparationAttempts });
            }
            if (!ClearScenery(preparationGoal) || !ClearCharacters(p, null, preparationGoal))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "search preparation destination became unsafe"));
            var ended = Time.unscaledTime - preparationLegAt >= 25
                ? Outcome.Failed(Wire.ReasonTimeout, "search travel leg time limit") : preparation.Tick(dt);
            if (ended == null) return null;
            preparation.Cancel(); preparation = null;
            if (ended.Status != Wire.StatusSucceeded)
            {
                Tracer.Emit("combat", "search_egress_rejected", new Dictionary<string, object>
                { ["goal"] = Convert.ToArray(preparationGoal), ["reason"] = ended.Reason, ["detail"] = ended.Detail });
                return null;
            }
            if (!Ground(pos, out var supported) || Math.Abs(pos.y - supported.y) > 1 || !ClearScenery(pos))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "search travel ended without clear terrain support"));
            preparing = false; area = supported;
            scouting = new Scouting(Convert.ToV3(origin), Convert.ToV3(area), Convert.ToV3(pos), Time.unscaledTime, searchRadius);
            Mode("search", "reached open ground through ordinary checked travel");
            return null;
        }

        Outcome Scout(Player p)
        {
            var pos = p.transform.position;
            if (Time.unscaledTime >= nextScout)
            {
                nextScout = Time.unscaledTime + .5f;
                scoutAction = scouting.Decide(Convert.ToV3(pos), p.GetStamina(), ClearCharacters(p, null, pos),
                    Time.unscaledTime, goal => ScoutRoute(p, goal));
                if (scoutAction == ScoutAction.Abort) return Abort(p, scouting.Reason);
                if (scoutAction == ScoutAction.Done)
                    return Finish(scouting.CompletedLegs == 0
                        ? Outcome.Failed(Wire.ReasonUnreachable, "no completed scouting route; creature search has no coverage")
                        : Outcome.Failed(Wire.ReasonInvalidTarget, "no eligible single Greyling after bounded local scouting and return"));
                if (scoutAction == ScoutAction.Wait) { walking = false; Mode("search"); return null; }
                var goal = Convert.ToVector3(scouting.Goal);
                if (!walking || Vector3.Distance(walkGoal, goal) > .2f)
                {
                    walkGoal = goal; walking = true;
                    walker.SetGoal(pos, goal, 1); walker.ResetStuck(pos);
                    Tracer.Emit("combat", "scout_route", new Dictionary<string, object>
                    { ["goal"] = Convert.ToArray(goal), ["returning"] = scouting.Returning, ["attempts"] = scouting.Attempts });
                }
                Mode(scouting.Returning ? "scout_return" : "scout");
            }
            if (scoutAction != ScoutAction.Walk) return null;
            if (walker.Refused() is Outcome refused) return Abort(p, refused.Detail ?? refused.Reason);
            if (!walker.Steer(pos, false, climbObstacles: false)) return Abort(p, "cannot follow the scouting route");
            return null;
        }

        V3? ScoutRoute(Player p, V3 desired)
        {
            string rejected = "wet_or_unloaded_terrain";
            if (!Ground(Convert.ToVector3(desired), out var goal)) return RejectScout(desired, rejected);
            approachPath.Clear();
            if (Pathfinding.instance == null || !Pathfinding.instance.GetPath(p.transform.position, goal, approachPath,
                Pathfinding.AgentType.Humanoid, true, true, false) || approachPath.Count == 0 || approachPath.Count > 32 ||
                Vector3.Distance(approachPath[approachPath.Count - 1], goal) > 1.5f)
                return RejectScout(desired, "no_complete_navmesh_route");
            probe.BeginSlice();
            if (!Ground(p.transform.position, out var supportedStart)) return RejectScout(desired, "unsupported_start");
            var previous = Convert.ToV3(supportedStart);
            double length = 0;
            // Include the exact destination after the navmesh endpoint.
            for (var i = 0; i <= approachPath.Count; i++)
            {
                var point = i == approachPath.Count ? goal : approachPath[i];
                if (!Ground(point, out var supported) || Math.Abs(point.y - supported.y) > 1)
                    return RejectScout(desired, "unsupported_navmesh_corner");
                var next = Convert.ToV3(supported);
                length += (next - previous).Length;
                if (length > 24 || Vector3.Distance(supported, origin) > searchRadius + .25)
                    return RejectScout(desired, "scouting_route_limit");
                foreach (var c in Character.GetAllCharacters())
                {
                    if (c == null || c == p || c.IsDead()) continue;
                    if (Scouting.RouteDistance(previous, next, Convert.ToV3(c.transform.position)) < (BaseAI.IsEnemy(p, c) ? 15 : 8))
                        return RejectScout(desired, "character_near_route");
                }
                // A 16 m scouting leg can be one straight navmesh edge;
                // the local corridor checker accepts at most 12 m per call.
                var parts = Math.Max(1, (int)Math.Ceiling((next - previous).Length / 8));
                var segment = previous;
                for (var part = 1; part <= parts; part++)
                {
                    var endpoint = previous + (next - previous) * ((double)part / parts);
                    if (!Ground(Convert.ToVector3(endpoint), out var supportedEnd)) return RejectScout(desired, "unsupported_navmesh_corner");
                    endpoint = Convert.ToV3(supportedEnd);
                    var failure = (endpoint - segment).Length < .05 ? null : GroundCorridor.Inspect(segment, endpoint, DryProbe);
                    if (failure != null) return RejectScout(desired, failure.Reason);
                    segment = endpoint;
                }
                if (!ClearScenery(supported)) return RejectScout(desired, "structure_near_route");
                previous = next;
            }
            return Convert.ToV3(goal);
        }

        V3? RejectScout(V3 goal, string why)
        {
            Tracer.Emit("combat", "scout_rejected", new Dictionary<string, object>
            { ["goal"] = goal.ToArray(), ["reason"] = why });
            return null;
        }

        V3? DryProbe(V3 from, V3 desired)
        {
            var result = probe.Probe(from, desired);
            if (!result.HasValue) return null;
            return ZoneSystem.instance != null && result.Value.Y > ZoneSystem.instance.m_waterLevel + .2f
                ? result : probe.Refuse("wet_ground", result.Value);
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
            groundRegion = null; escapeEnd = null;
            probe.BeginSlice();
            rejected = "wet_or_unloaded_terrain";
            if (!Ground(at, out grounded)) return false;
            rejected = "different_ground_level";
            if (Math.Abs(at.y - grounded.y) > 1) return false;
            var center = Convert.ToV3(grounded);
            V3? Project(V3 point) => Ground(Convert.ToVector3(point), out var support) ? Convert.ToV3(support) : (V3?)null;
            groundFailure = FightingGround.Inspect(center, Project, DryProbe, out groundRegion, out escapeEnd);
            if (groundFailure != null) { rejected = groundFailure.Reason; return false; }
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
            if (groundRegion != null) data["region"] = groundRegion;
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
            if (physical != null && groundFailure?.Reason == "blocked_or_unsupported")
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

        Outcome Finish(Outcome result) { result.Entity = string.IsNullOrEmpty(entity) ? null : entity; Cancel(); return result; }
        public void Cancel()
        {
            cancelled = true; melee?.Cancel(); escape?.Cancel(); preparation?.Cancel();
            if (ReferenceEquals(Active, this)) Active = null;
            Driver.ClearCombat();
        }
    }
}
