using System;

namespace Renki.Movement
{
    /// <summary>Bounded escape guidance. Prefix checks choose a bearing;
    /// the normal walker must still validate and execute the actual route.</summary>
    public sealed class FleeBearings
    {
        static readonly int[] angles = { 0, 30, -30, 60, -60, 90, -90 };
        public const int CandidateLimit = 7, PrefixSteps = 6;
        readonly Func<V3, V3, V3?> probe;
        readonly double distance;
        V3 origin, threat, supported;
        int index, part;
        public RouteStatus Status { get; private set; } = RouteStatus.Searching;
        public int Samples { get; private set; }
        public int Candidate => index;
        public int Degrees => angles[Math.Min(index, CandidateLimit - 1)];
        public V3 Goal => AtBearing(origin, threat, distance, Degrees);

        public FleeBearings(V3 from, V3 threat, double distance, Func<V3, V3, V3?> probe)
        {
            origin = supported = from; this.threat = threat;
            this.distance = distance; this.probe = probe;
        }

        public static V3 AtBearing(V3 from, V3 threat, double distance, int degrees)
        {
            var away = (from - threat).Flat.Normalized;
            if (away.Length < .5) away = V3.Forward;
            var angle = degrees * Math.PI / 180;
            var cosine = Math.Abs(degrees) == 90 ? 0 : Math.Cos(angle);
            var sine = Math.Sin(angle);
            var direction = new V3(away.X * cosine - away.Z * sine, 0,
                away.X * sine + away.Z * cosine);
            var gap = V3.FlatDistance(from, threat);
            var radius = Math.Max(distance, gap) + Flee.Beyond;
            var length = -gap * cosine + Math.Sqrt(Math.Max(0, radius * radius - gap * gap * sine * sine));
            return from + direction * length;
        }

        public RouteStatus Advance(int budget = 8)
        {
            for (var n = 0; n < budget && Status == RouteStatus.Searching; n++)
            {
                var direction = (Goal - origin).Flat.Normalized;
                var desired = origin + direction * ((part + 1) * .5);
                var at = probe(supported, desired);
                Samples++;
                if (!at.HasValue || !Finite(at.Value) || V3.FlatDistance(at.Value, desired) > .1)
                { Next(); continue; }
                supported = at.Value;
                if (++part == PrefixSteps) Status = RouteStatus.Found;
            }
            return Status;
        }

        // Keep a tangent target anchored while the bot moves toward it.
        // Recomputing it from every new body position would circle a static
        // threat instead of obtaining the requested separation.
        public V3 Reaim(V3 from, V3 currentThreat, double after = 3)
        {
            if (Status == RouteStatus.Found && V3.FlatDistance(threat, currentThreat) >= after)
            { origin = from; threat = currentThreat; }
            return Goal;
        }

        // A refused goal consumes its bearing for this entire flee child.
        // Re-aiming or making progress never replenishes that finite budget.
        public void Reject(V3 from, V3 currentThreat)
        {
            if (Status != RouteStatus.Found) return;
            origin = from; threat = currentThreat;
            Next();
        }

        void Next()
        {
            part = 0; supported = origin;
            Status = ++index < CandidateLimit ? RouteStatus.Searching : RouteStatus.NoRoute;
        }

        public void Cancel() => Status = RouteStatus.NoRoute;
        static bool Finite(V3 p) => !double.IsNaN(p.X) && !double.IsInfinity(p.X) &&
            !double.IsNaN(p.Y) && !double.IsInfinity(p.Y) && !double.IsNaN(p.Z) && !double.IsInfinity(p.Z);
    }
}
