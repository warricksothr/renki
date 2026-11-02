using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    /// <summary>Reuse identical support queries, never merge storeys or
    /// nearby heights. A scene revision or short expiry forces a new probe.</summary>
    public sealed class SupportMemo
    {
        readonly Dictionary<(double, double, double), (double At, V3? Value, object Evidence)> entries = new Dictionary<(double, double, double), (double, V3?, object)>();
        readonly Func<V3, double, (V3? Point, object Evidence)> probe;
        long revision = -1;
        public int Hits { get; private set; }
        public int Misses { get; private set; }
        public int Count => entries.Count;
        public object LastEvidence { get; private set; }
        public SupportMemo(Func<V3, double, V3?> probe) : this((at, y) => (probe(at, y), null)) { }
        public SupportMemo(Func<V3, double, (V3? Point, object Evidence)> probe) { this.probe = probe; }
        public V3? Sample(V3 at, double fromY, double now, long sceneRevision)
        {
            if (revision != sceneRevision) { entries.Clear(); revision = sceneRevision; }
            var key = (at.X, at.Z, fromY);
            if (entries.TryGetValue(key, out var old) && now >= old.At && now - old.At < 1)
            { Hits++; LastEvidence = old.Evidence; return old.Value; }
            if (entries.Count >= 8192) entries.Clear();
            Misses++;
            var value = probe(at, fromY);
            entries[key] = (now, value.Point, value.Evidence);
            LastEvidence = value.Evidence;
            return value.Point;
        }
    }
}
