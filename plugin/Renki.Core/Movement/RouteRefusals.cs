using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>Short-lived negative evidence from movement, shared by replans
    /// of one goal. It never authorizes movement or blocks backing away.</summary>
    public sealed class RouteRefusals
    {
        public const int Limit = 32;
        readonly List<Entry> entries = new List<Entry>();
        sealed class Entry
        {
            public V3 From, To;
            public double Until;
            public Func<bool> Current;
        }
        public int Count => entries.Count;
        public void Clear() => entries.Clear();
        void Sweep(double now) => entries.RemoveAll(e => now >= e.Until || !e.Current());
        public void Remember(V3 from, V3 to, double now, Func<bool> current)
        {
            if ((to - from).Flat.Length < .01) return;
            Sweep(now);
            entries.RemoveAll(e => (e.From - from).Length < .25 && (e.To - to).Length < .25);
            if (entries.Count >= Limit) entries.RemoveAt(0);
            entries.Add(new Entry { From = from, To = to, Until = now + 45, Current = current });
        }
        public bool Blocks(V3 from, V3 to, double now)
        {
            Sweep(now);
            foreach (var e in entries)
            {
                if (Math.Abs(from.Y - e.From.Y) > .6 || Math.Abs(to.Y - e.To.Y) > .6) continue;
                if (V3.Dot((to - from).Flat, (e.To - e.From).Flat) <= 0) continue;
                WallRoute.Closest(from.Flat, to.Flat, e.To.Flat, e.To.Flat, out var t, out _);
                if (V3.FlatDistance(from + (to - from) * t, e.To) < .35) return true;
            }
            return false;
        }
    }
}
