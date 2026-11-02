using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>Try the ordinary checked search before one guided fallback.
    /// Terrain hints cannot displace a bridge that the first search finds.</summary>
    public sealed class RouteSearch
    {
        readonly Func<ThreatGuidance, LocalRoute> create;
        readonly Func<ThreatGuidance> guidance;
        LocalRoute route;
        int previousExpanded, previousProbes;
        bool tried;
        public ThreatGuidance Guidance { get; private set; }
        public int Phase { get; private set; }
        public int Expanded => previousExpanded + route.Expanded;
        public int Probes => previousProbes + route.Probes;
        public bool NodeLimitReached => route.NodeLimitReached;
        public RouteStatus Status => route.Status;
        public IReadOnlyList<V3> Path => route.Path;
        public bool ProgressLeg => route.ProgressLeg;

        public RouteSearch(Func<ThreatGuidance, LocalRoute> create, Func<ThreatGuidance> guidance, bool preferGuidance = false)
        {
            this.create = create ?? throw new ArgumentNullException(nameof(create));
            this.guidance = guidance;
            if (preferGuidance) { tried = true; Guidance = guidance?.Invoke(); }
            route = create(Guidance != null && Guidance.Active ? Guidance : null);
        }

        public RouteStatus Advance(int budget = 8)
        {
            route.Advance(budget);
            if (route.Status != RouteStatus.NoRoute || tried) return route.Status;
            tried = true;
            var next = guidance?.Invoke();
            if (next == null || !next.Active) return route.Status;
            previousExpanded = route.Expanded; previousProbes = route.Probes;
            Guidance = next; Phase++;
            route = create(next);
            return route.Status;
        }
    }
}
