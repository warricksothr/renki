namespace Renki.Movement
{
    /// <summary>Physical escape progress, independent of goals and replans.</summary>
    public sealed class FleeProgress
    {
        V3 anchor;
        double since;
        bool watching;
        public bool Blocked(V3 pos, double gap, double now)
        {
            if (gap > 15) { watching = false; return false; }
            if (!watching || V3.FlatDistance(pos, anchor) >= 1)
            { watching = true; anchor = pos; since = now; return false; }
            return now - since >= 4;
        }
    }
}
