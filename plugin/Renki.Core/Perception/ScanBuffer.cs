using System;

namespace Renki.Perception
{
    /// <summary>A full nonalloc query is ambiguous: grow and query again,
    /// never treat a truncated prefix as a complete safety observation.</summary>
    public sealed class ScanBuffer<T>
    {
        readonly int maximum;
        public T[] Items { get; private set; }
        public int Count { get; private set; }

        public ScanBuffer(int initial, int maximum)
        {
            if (initial <= 0 || maximum < initial) throw new ArgumentOutOfRangeException(nameof(initial));
            this.maximum = maximum;
            Items = new T[initial];
        }

        public bool TryFill(Func<T[], int> query)
        {
            while (true)
            {
                Count = query(Items);
                if (Count < 0 || Count > Items.Length) return false;
                if (Count < Items.Length) return true;
                if (Items.Length == maximum) return false;
                Items = new T[(int)Math.Min(maximum, (long)Items.Length * 2)];
            }
        }
    }
}
