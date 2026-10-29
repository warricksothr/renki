using System.Collections.Generic;

namespace Renki.Perception
{
    /// <summary>
    /// One kind of thing in a perception snapshot (docs/BRIDGE-PROTOCOL.md
    /// 6.2): everything in range goes in, and the nearest Cap come out,
    /// nearest first. Reused from scan to scan, so it allocates only while
    /// growing.
    /// </summary>
    public sealed class Nearest<T>
    {
        readonly List<KeyValuePair<double, T>> items = new List<KeyValuePair<double, T>>();
        static readonly IComparer<KeyValuePair<double, T>> ByDistance =
            Comparer<KeyValuePair<double, T>>.Create((a, b) => a.Key.CompareTo(b.Key));

        public int Cap;

        public Nearest(int cap)
        {
            Cap = cap;
        }

        /// <summary>How many went in since the last Clear, before the cap.</summary>
        public int Seen => items.Count;

        public void Clear() => items.Clear();

        public void Add(T item, double distance) => items.Add(new KeyValuePair<double, T>(distance, item));

        /// <summary>The nearest Cap, nearest first.</summary>
        public List<T> Take()
        {
            items.Sort(ByDistance);
            var n = System.Math.Min(System.Math.Max(Cap, 0), items.Count);
            var nearest = new List<T>(n);
            for (var i = 0; i < n; i++) nearest.Add(items[i].Value);
            return nearest;
        }
    }

    /// <summary>
    /// Nearest per kind: each kind (tree, rock, drop...) keeps its own
    /// nearest Cap, so a forest cannot crowd out the wood lying on the
    /// ground. Take merges them, nearest first.
    /// </summary>
    public sealed class NearestByKind<T>
    {
        readonly Dictionary<string, Nearest<KeyValuePair<double, T>>> kinds = new Dictionary<string, Nearest<KeyValuePair<double, T>>>();
        static readonly IComparer<KeyValuePair<double, T>> ByDistance =
            Comparer<KeyValuePair<double, T>>.Create((a, b) => a.Key.CompareTo(b.Key));

        public int Cap;

        public NearestByKind(int cap)
        {
            Cap = cap;
        }

        /// <summary>How many went in since the last Clear, before the caps.</summary>
        public int Seen { get; private set; }

        public void Clear()
        {
            foreach (var k in kinds.Values) k.Clear();
            Seen = 0;
        }

        public void Add(string kind, T item, double distance)
        {
            if (!kinds.TryGetValue(kind, out var k)) kinds[kind] = k = new Nearest<KeyValuePair<double, T>>(Cap);
            k.Add(new KeyValuePair<double, T>(distance, item), distance);
            Seen++;
        }

        /// <summary>The nearest Cap of each kind, nearest first overall.</summary>
        public List<T> Take()
        {
            var all = new List<KeyValuePair<double, T>>();
            foreach (var k in kinds.Values)
            {
                k.Cap = Cap;
                all.AddRange(k.Take());
            }
            all.Sort(ByDistance);
            var nearest = new List<T>(all.Count);
            foreach (var kv in all) nearest.Add(kv.Value);
            return nearest;
        }
    }
}
