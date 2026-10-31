using System;
using System.Collections.Generic;
using Renki.Movement;

namespace Renki.Tasks
{
    /// <summary>A finite local plan, never permission to move or replay old ground.</summary>
    public sealed class ScoutDetour
    {
        public const int NodeLimit = 1024, SliceLimit = 8, AttemptLimit = 4;
        public const double TimeLimit = 4, PathLimit = 24;
        readonly LocalRoute route;
        readonly double began;
        public V3 Origin { get; }
        public V3 Target { get; }
        public RouteStatus Status { get; private set; } = RouteStatus.Searching;
        public string Reason { get; private set; }
        public int Expanded => route.Expanded;
        public int Probes => route.Probes;
        public IReadOnlyList<V3> Path => Status == RouteStatus.Found ? route.Path : null;

        public ScoutDetour(V3 start, V3 target, Func<V3, V3, V3?> probe, double now)
        {
            if (!Finite(now) || !Finite(start.X) || !Finite(start.Y) || !Finite(start.Z) ||
                !Finite(target.X) || !Finite(target.Y) || !Finite(target.Z)) throw new ArgumentOutOfRangeException(nameof(start));
            if (probe == null) throw new ArgumentNullException(nameof(probe));
            Origin = start; Target = target; began = now;
            route = new LocalRoute(start, target, probe, groundCorridor: true, maxCost: PathLimit, maxNodes: NodeLimit);
        }

        public RouteStatus Advance(double now, int budget = 1)
        {
            if (budget < 1 || budget > SliceLimit) throw new ArgumentOutOfRangeException(nameof(budget));
            if (Status != RouteStatus.Searching) return Status;
            if (double.IsNaN(now) || double.IsInfinity(now) || now < began || now - began >= TimeLimit)
                return Stop("planning deadline");
            Status = route.Advance(budget);
            if (Status == RouteStatus.NoRoute) Reason = "no checked local detour within bounds";
            return Status;
        }

        public void Cancel() { Stop("cancelled"); }
        static bool Finite(double value) => !double.IsNaN(value) && !double.IsInfinity(value);
        RouteStatus Stop(string reason) { Reason = reason; Status = RouteStatus.NoRoute; return Status; }
    }
}
