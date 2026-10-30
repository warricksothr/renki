using System;

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
        readonly V3 origin, area;
        readonly double began, radius;
        V3 last, destination;
        double progressAt, bestDistance;
        bool hasGoal, hasDestination, ending;
        public V3 Goal { get; private set; }
        public bool Returning { get; private set; }
        public int Attempts { get; private set; }
        public int CompletedLegs { get; private set; }
        public double Travel { get; private set; }
        public string Reason { get; private set; }

        public Scouting(V3 origin, V3 area, V3 bot, double now, double radius = 60)
        {
            if (!Finite(radius) || radius < 12 || radius > Leash) throw new ArgumentOutOfRangeException(nameof(radius));
            this.origin = origin; this.area = area; this.radius = radius;
            last = bot; began = progressAt = now;
        }

        public ScoutAction Decide(V3 bot, double stamina, bool safe, double now, Func<V3, V3?> route)
        {
            Reason = null;
            if (!safe) return Abort("another character is too close to scout safely");
            if (!Finite(bot) || !Finite(stamina) || !Finite(now)) return Abort("invalid scouting state");
            Travel += (bot - last).Length; last = bot;
            if (!WithinBounds(bot) || Travel > TravelLimit) return Abort("scouting travel limit");
            if (now - began >= ReturnLimit) return Abort("scouting return time limit");
            ending |= now - began >= WorkLimit || stamina < 30 || Travel >= TravelLimit - radius * 1.5 ||
                Attempts >= CandidateLimit && (!hasDestination || V3.FlatDistance(bot, destination) <= 1.5);
            if (ending && !Returning)
            {
                Returning = true; hasGoal = false; hasDestination = true; destination = area;
            }
            if (Returning && (bot - area).Length <= 1.5) return ScoutAction.Done;
            if (hasGoal && (bot - Goal).Length <= 1.5) { CompletedLegs++; hasGoal = false; }
            if (!hasGoal)
            {
                if (!hasDestination || V3.FlatDistance(bot, destination) <= 1.5)
                {
                    if (now - began < 2) return ScoutAction.Wait;
                    if (Attempts >= CandidateLimit) return ScoutAction.Done;
                    destination = Candidate(origin, Attempts++, radius); hasDestination = true;
                }
                // Only a short, physically checked leg is committed. More
                // distant terrain can load before the next leg is considered.
                var offset = (destination - bot).Flat;
                var desired = offset.Length <= LegLength ? destination : bot + offset.Normalized * LegLength;
                var supported = route(desired);
                if (!supported.HasValue || !WithinBounds(supported.Value))
                {
                    if (Returning) return Abort("cannot return along a checked scouting route");
                    hasDestination = false;
                    return ScoutAction.Wait;
                }
                Goal = supported.Value; hasGoal = true;
                bestDistance = (bot - Goal).Length; progressAt = now;
            }
            else
            {
                var supported = route(Goal);
                if (!supported.HasValue || !WithinBounds(supported.Value) || (supported.Value - Goal).Length > .65)
                    return Abort("scouting route changed or became unsafe");
            }
            var distance = (bot - Goal).Length;
            if (distance <= bestDistance - .5) { bestDistance = distance; progressAt = now; }
            if (now - progressAt >= 4) return Abort("no progress along the scouting route");
            return ScoutAction.Walk;
        }

        bool WithinBounds(V3 at) => Finite(at) && (at - origin).Length <= radius + .25;
        ScoutAction Abort(string reason) { Reason = reason; return ScoutAction.Abort; }
        static bool Finite(V3 at) => Finite(at.X) && Finite(at.Y) && Finite(at.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);

        public static V3 Candidate(V3 origin, int attempt, double radius = 60)
        {
            if (attempt < 0 || attempt >= CandidateLimit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var angle = attempt / 3 * Math.PI / 4;
            return origin + new V3(Math.Cos(angle), 0, Math.Sin(angle)) * (radius * (attempt % 3 + 1) / 3);
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
