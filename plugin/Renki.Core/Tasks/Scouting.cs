using System;

namespace Renki.Tasks
{
    public enum ScoutAction { Wait, Walk, Done, Abort }

    /// <summary>Finite out-and-back search. The host checks routes and acquires
    /// enemies; this policy never chooses a target or bypasses movement checks.</summary>
    public sealed class Scouting
    {
        public const int CandidateLimit = 16;
        public const double WorkLimit = 25, ReturnLimit = 35, Leash = 12, TravelLimit = 65;
        readonly V3 origin, area;
        readonly double began;
        V3 last;
        double progressAt, bestDistance;
        bool hasGoal, ending;
        public V3 Goal { get; private set; }
        public bool Returning { get; private set; }
        public int Attempts { get; private set; }
        public double Travel { get; private set; }
        public string Reason { get; private set; }

        public Scouting(V3 origin, V3 area, V3 bot, double now)
        { this.origin = origin; this.area = area; last = bot; began = progressAt = now; }

        public ScoutAction Decide(V3 bot, double stamina, bool safe, double now, Func<V3, V3?> route)
        {
            Reason = null;
            Travel += (bot - last).Length; last = bot;
            if (!safe) return Abort("another character is too close to scout safely");
            if (!Finite(bot) || !Finite(stamina) || !Finite(now)) return Abort("invalid scouting state");
            if ((bot - origin).Length > Kiting.TravelLimit || (bot - area).Length > Leash || Travel > TravelLimit)
                return Abort("scouting travel limit");
            if (now - began >= ReturnLimit) return Abort("scouting return time limit");
            ending |= now - began >= WorkLimit || stamina < 30 || Travel >= 50 || Attempts >= CandidateLimit;
            if (ending && (!hasGoal || !Returning)) SetGoal(area, bot, now, true);
            if (hasGoal && (bot - Goal).Length <= 1.5)
            {
                if (!Returning) SetGoal(area, bot, now, true);
                else
                {
                    hasGoal = false;
                    if (ending || Attempts >= CandidateLimit) return ScoutAction.Done;
                }
            }
            if (!hasGoal)
            {
                if (now - began < 2) return ScoutAction.Wait;
                if (Attempts >= CandidateLimit) return ScoutAction.Done;
                var supported = route(Candidate(area, Attempts++));
                if (!supported.HasValue) return ScoutAction.Wait;
                if (!WithinBounds(supported.Value)) return Abort("scouting route left the local search area");
                SetGoal(supported.Value, bot, now, false);
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

        void SetGoal(V3 goal, V3 bot, double now, bool returning)
        { Goal = goal; hasGoal = true; Returning = returning; bestDistance = (bot - goal).Length; progressAt = now; }

        bool WithinBounds(V3 at) => Finite(at) && (at - area).Length <= Leash && (at - origin).Length <= Kiting.TravelLimit;
        ScoutAction Abort(string reason) { Reason = reason; return ScoutAction.Abort; }
        static bool Finite(V3 at) => Finite(at.X) && Finite(at.Y) && Finite(at.Z);
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);

        public static V3 Candidate(V3 area, int attempt)
        {
            if (attempt < 0 || attempt >= CandidateLimit) throw new ArgumentOutOfRangeException(nameof(attempt));
            var angle = attempt % 8 * Math.PI / 4;
            return area + new V3(Math.Cos(angle), 0, Math.Sin(angle)) * (attempt < 8 ? 6 : 10);
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
