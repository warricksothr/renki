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
        bool tried, incremental;
        public bool BuildingGuidance { get; private set; }
        // Incremental graph construction gets its own finite wall-time stage.
        public int BudgetStages => Phase + 1 + (incremental ? 1 : 0);
        public ThreatGuidance Guidance { get; private set; }
        public int Phase { get; private set; }
        public int Expanded => previousExpanded + route.Expanded;
        public int Probes => previousProbes + route.Probes;
        public bool NodeLimitReached => route.NodeLimitReached;
        public RouteStatus Status => BuildingGuidance ? RouteStatus.Searching : route.Status;
        public IReadOnlyList<V3> Path => route.Path;
        public bool ProgressLeg => route.ProgressLeg;

        public RouteSearch(Func<ThreatGuidance, LocalRoute> create, Func<ThreatGuidance> guidance, bool preferGuidance = false)
        {
            this.create = create ?? throw new ArgumentNullException(nameof(create));
            this.guidance = guidance;
            if (preferGuidance) { tried = true; Guidance = guidance?.Invoke(); }
            BuildingGuidance = Guidance != null && !Guidance.Complete;
            incremental = BuildingGuidance;
            route = create(Guidance != null && Guidance.Complete && Guidance.Active ? Guidance : null);
        }

        public RouteStatus Advance(int budget = 8)
        {
            if (BuildingGuidance)
            {
                if (!Guidance.Advance(budget)) return RouteStatus.Searching;
                BuildingGuidance = false;
                UseGuidance();
                return route.Status;
            }
            route.Advance(budget);
            if (route.Status != RouteStatus.NoRoute || tried) return route.Status;
            tried = true;
            Guidance = guidance?.Invoke();
            if (Guidance != null && !Guidance.Complete)
            {
                incremental = BuildingGuidance = true;
                return RouteStatus.Searching;
            }
            UseGuidance();
            return route.Status;
        }

        void UseGuidance()
        {
            if (Guidance == null || !Guidance.Active) return;
            previousExpanded = route.Expanded; previousProbes = route.Probes;
            if (route.Status == RouteStatus.NoRoute) Phase++;
            route = create(Guidance);
        }
    }
}
