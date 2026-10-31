using System;
using System.Collections.Generic;

namespace Renki.Tasks
{
    public enum ScoutAction { Wait, Walk, Done, Abort }

    /// <summary>Progressive local search with checked legs and a bounded return.
    /// Acquisition and all physical route validation remain with the host.</summary>
    public sealed class Scouting
    {
        public const int CandidateLimit = 24;
        public const double WorkLimit = 90, ReturnLimit = 150, Leash = 80, TravelLimit = 300;
        public const double LegLength = 16;
        public const double OuterReach = 120;
        public const int LegAlternatives = 4;
        public const int RecoveryLimit = 4, TrailLimit = 160;
        const double TrailSpacing = 2;
        readonly V3 origin, area;
        readonly double began, radius, offset;
        readonly int firstCandidate, candidateCount;
        readonly List<V3> trail = new List<V3>();
        V3 last, destination;
        double progressAt, bestDistance;
        bool hasGoal, hasDestination, ending;
        int legAlternative;
        public V3 Goal { get; private set; }
        public bool Returning { get; private set; }
        public int Attempts { get; private set; }
        public int CompletedLegs { get; private set; }
        public int Recoveries { get; private set; }
        public int TrailPoints => trail.Count;
        // These are return hints from actual observations, never authority to
        // walk. Every connector still goes through the current route validator.
        public double ReturnDistance
        {
            get
            {
                double length = 0;
                var previous = last;
                for (var i = trail.Count - 1; i >= 0; i--)
                {
                    length += (previous - trail[i]).Length;
                    previous = trail[i];
                }
                return length;
            }
        }
        public double Travel { get; private set; }
        public string Reason { get; private set; }

        public Scouting(V3 origin, V3 area, V3 bot, double now, double radius = 60, int? sector = null, double offset = 0)
        {
            if (!Finite(radius) || radius < 12 || radius > Leash) throw new ArgumentOutOfRangeException(nameof(radius));
            if (sector.HasValue && (sector < 0 || sector > 7)) throw new ArgumentOutOfRangeException(nameof(sector));
            if (!Finite(offset) || offset < 0 || offset > 60 || radius + offset > OuterReach || offset > 0 && !sector.HasValue)
                throw new ArgumentOutOfRangeException(nameof(offset));
            firstCandidate = (sector ?? 0) * 3;
            candidateCount = sector.HasValue ? 3 : CandidateLimit;
            this.origin = origin; this.area = area; this.radius = radius; this.offset = offset;
            last = bot; began = progressAt = now;
            trail.Add(area);
        }

        public ScoutAction Decide(V3 bot, double stamina, bool safe, double now, Func<V3, V3?> route)
        {
            Reason = null;
            if (!safe) return Abort("another character is too close to scout safely");
            if (!Finite(bot) || !Finite(stamina) || !Finite(now)) return Abort("invalid scouting state");
            Travel += (bot - last).Length; last = bot;
            if (!WithinBounds(bot) || Travel > TravelLimit) return Abort("scouting travel limit");
            if (now - began >= ReturnLimit) return Abort("scouting return time limit");
            if (!Returning && hasGoal && (bot - trail[trail.Count - 1]).Length >= TrailSpacing)
            {
                if (trail.Count < TrailLimit) trail.Add(bot);
                else ending = true;
            }
            if (hasGoal && (bot - Goal).Length <= 1.5)
            {
                if (!Returning) CompletedLegs++;
                hasGoal = false;
            }
            ending |= now - began >= WorkLimit || stamina < 30 || Travel + ReturnDistance >= TravelLimit - LegLength ||
                Attempts >= candidateCount && (!hasDestination || V3.FlatDistance(bot, destination) <= 1.5);
            // Longer outbound trips reserve time for the actual winding trail
            // and half-second breadcrumb decisions before the existing hard
            // return deadline. This is a planning allowance, never permission
            // to replay the trail without checking it again.
            ending |= offset > 0 && now - began + ReturnDistance / 1.5 + trail.Count * .5 >= ReturnLimit - 10;
            if (ending && !Returning)
            {
                Returning = true; hasGoal = false; hasDestination = false;
            }
            if (Returning && (bot - area).Length <= 1.5) return ScoutAction.Done;
            if (!hasGoal)
            {
                if (Returning)
                {
                    // Revisit the actual winding trail instead of inventing a
                    // straight return across ground we never traversed.
                    while (trail.Count > 1 && (bot - trail[trail.Count - 1]).Length <= 1.5)
                        trail.RemoveAt(trail.Count - 1);
                    destination = trail[trail.Count - 1]; hasDestination = true;
                }
                else if (!hasDestination || V3.FlatDistance(bot, destination) <= 1.5)
                {
                    if (now - began < 2) return ScoutAction.Wait;
                    if (Attempts >= candidateCount) return ScoutAction.Done;
                    destination = Candidate(origin, firstCandidate + Attempts++, radius, offset); hasDestination = true;
                    legAlternative = 0;
                }
                // Only a short, physically checked leg is committed. More
                // distant terrain can load before the next leg is considered.
                var delta = (destination - bot).Flat;
                var desired = Returning || this.offset == 0
                    ? delta.Length <= LegLength ? destination : bot + delta.Normalized * LegLength
                    : LegCandidate(bot, destination, legAlternative);
                var supported = route(desired);
                if (!supported.HasValue || !WithinBounds(supported.Value) ||
                    !Returning && this.offset > 0 && V3.FlatDistance(supported.Value, destination) >= V3.FlatDistance(bot, destination) - .1)
                {
                    if (Returning) return Abort("cannot return along a checked scouting route");
                    if (this.offset > 0)
                    {
                        // One probe per decision: spread the finite local fan
                        // across ticks instead of doing several physics scans.
                        if (++legAlternative < LegAlternatives) return ScoutAction.Wait;
                        return RejectRoute("no checked progress toward outer search region");
                    }
                    hasDestination = false;
                    return ScoutAction.Wait;
                }
                Goal = supported.Value; hasGoal = true; legAlternative = 0;
                bestDistance = (bot - Goal).Length; progressAt = now;
            }
            else
            {
                var supported = route(Goal);
                if (!supported.HasValue || !WithinBounds(supported.Value) || (supported.Value - Goal).Length > .65)
                    return RejectRoute("scouting route changed or became unsafe");
            }
            var distance = (bot - Goal).Length;
            if (distance <= bestDistance - .5) { bestDistance = distance; progressAt = now; }
            if (now - progressAt >= 4) return RejectRoute("no progress along the scouting route");
            return ScoutAction.Walk;
        }

        // Host Walker failures and policy revalidation failures share the same
        // finite allowance. No retry resets travel/time/candidate accounting.
        public ScoutAction RejectRoute(string reason)
        {
            if (Returning) return Abort(reason);
            Recoveries = Math.Min(RecoveryLimit, Recoveries + 1);
            hasGoal = hasDestination = false;
            // Advance to the next bearing instead of committing another
            // collinear leg toward a farther point on the rejected spoke.
            Attempts = Math.Min(candidateCount, (Attempts + 2) / 3 * 3);
            ending |= Recoveries >= RecoveryLimit || Attempts >= candidateCount;
            Reason = reason;
            return ScoutAction.Wait;
        }

        bool WithinBounds(V3 at) => Finite(at) && (at - origin).Length <= radius + offset + .25;
        ScoutAction Abort(string reason) { Reason = reason; return ScoutAction.Abort; }
        static bool Finite(V3 at) => Finite(at.X) && Finite(at.Y) && Finite(at.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);

        // Travel has no swings/acquisition. Combat uses its separate buffers.
        public static bool TravelSeparated(double distance, bool player, bool hostile, bool destination) =>
            Finite(distance) && distance >= (player ? destination ? 2 : 0 : hostile ? 15 : 8);

        public static V3 Candidate(V3 origin, int attempt, double radius = 60, double offset = 0)
        {
            if (attempt < 0 || attempt >= CandidateLimit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var angle = attempt / 3 * Math.PI / 4;
            return origin + new V3(Math.Cos(angle), 0, Math.Sin(angle)) * (offset + radius * (attempt % 3 + 1) / 3);
        }

        public static V3 LegCandidate(V3 bot, V3 destination, int alternative)
        {
            if (alternative < 0 || alternative >= LegAlternatives) throw new ArgumentOutOfRangeException(nameof(alternative));
            var delta = (destination - bot).Flat;
            var length = Math.Min(alternative == 3 ? LegLength / 2 : LegLength, delta.Length);
            var direction = delta.Normalized;
            var angle = alternative == 1 ? Math.PI / 6 : alternative == 2 ? -Math.PI / 6 : 0;
            return bot + new V3(direction.X * Math.Cos(angle) - direction.Z * Math.Sin(angle), 0,
                direction.X * Math.Sin(angle) + direction.Z * Math.Cos(angle)) * length;
        }

        public static double RouteDistance(V3 from, V3 to, V3 at)
        {
            var segment = to - from;
            var lengthSquared = V3.Dot(segment, segment);
            var fraction = lengthSquared > 1e-9 ? Math.Max(0, Math.Min(1, V3.Dot(at - from, segment) / lengthSquared)) : 0;
            return (at - (from + segment * fraction)).Length;
        }
    }
}
