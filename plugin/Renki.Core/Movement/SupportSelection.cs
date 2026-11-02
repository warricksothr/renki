using System;

namespace Renki.Movement
{
    /// <summary>The native ray's existing selection policy, independent of
    /// physics. Do not substitute a lower floor when the selected hit is
    /// outside the step band; report that ambiguity instead.</summary>
    public struct SupportSelection
    {
        readonly double fromY, step;
        readonly bool footing, walking;
        V3? selected;
        public int Hits { get; private set; }
        public int NormalHits { get; private set; }
        public V3? Selected => selected;
        public V3? Point => selected.HasValue && Math.Abs(selected.Value.Y - fromY) <= step ? selected : null;
        public string Refusal => Point.HasValue ? null : Hits == 0 ? "no_hits" : NormalHits == 0 ? "slope" :
            !selected.HasValue ? "footing_band" : "height_band";

        public SupportSelection(double fromY, double step, bool footing, bool walking) : this()
        { this.fromY = fromY; this.step = step; this.footing = footing; this.walking = walking; }

        public void Consider(V3 point, double normalY)
        {
            Hits++;
            if (!(walking ? DryTravel.WalkableNormal(normalY) : normalY >= .35f)) return;
            NormalHits++;
            if (footing ? DryTravel.PreferFooting(fromY, point.Y, selected?.Y) :
                !selected.HasValue || point.Y > selected.Value.Y) selected = point;
        }
    }
}
