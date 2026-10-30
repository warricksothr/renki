using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;

namespace Renki.Perception
{
    public readonly struct TerrainSample
    {
        public readonly double Height;
        public readonly string Biome;
        public TerrainSample(double height, string biome) { Height = height; Biome = biome; }
    }

    /// <summary>Bounded visual terrain cache. A step samples at most one tile
    /// (25 points). Near tiles lead; equally distant tiles ahead lead those
    /// behind. Missing heightmaps retry after 5 s, known tiles after 30 s.
    /// Instances belong to one world; observations do not prove walkability.</summary>
    public sealed class TerrainCache
    {
        public const int TileSize = 16, Spacing = 4, Capacity = 256;
        sealed class Entry
        {
            public TerrainTile Tile;
            public double Due;
            public bool Sent;
        }
        readonly Dictionary<(int, int), Entry> entries = new Dictionary<(int, int), Entry>();
        public int Count => entries.Count;
        public long Samples { get; private set; }
        public long Builds { get; private set; }
        public long Changes { get; private set; }

        public void Resend() { foreach (var e in entries.Values) e.Sent = false; }

        public void Invalidate(double minX, double minZ, double maxX, double maxZ)
        {
            foreach (var kv in entries)
                if ((kv.Key.Item1 + 1) * TileSize >= minX && kv.Key.Item1 * TileSize <= maxX &&
                    (kv.Key.Item2 + 1) * TileSize >= minZ && kv.Key.Item2 * TileSize <= maxZ)
                    kv.Value.Due = double.NegativeInfinity;
        }

        public TerrainTile Step(double x, double z, double forwardX, double forwardZ, double now,
            double water, Func<double, double, TerrainSample?> sample, int radius = 3)
        {
            radius = Math.Max(1, Math.Min(4, radius));
            var cx = (int)Math.Floor(x / TileSize); var cz = (int)Math.Floor(z / TileSize);
            (int, int) chosen = default;
            var best = double.PositiveInfinity;
            Entry old = null;
            for (var dz = -radius; dz <= radius; dz++)
                for (var dx = -radius; dx <= radius; dx++)
                {
                    var key = (cx + dx, cz + dz);
                    entries.TryGetValue(key, out var e);
                    if (e != null && e.Sent && e.Due > now && e.Tile.WaterLevel == water) continue;
                    var tx = (key.Item1 + .5) * TileSize - x; var tz = (key.Item2 + .5) * TileSize - z;
                    var rank = tx * tx + tz * tz - (tx * forwardX + tz * forwardZ) * 4;
                    if (rank >= best) continue;
                    best = rank; chosen = key; old = e;
                }
            if (double.IsPositiveInfinity(best)) return null;
            if (old != null && !old.Sent && old.Due > now && old.Tile.WaterLevel == water)
            {
                old.Sent = true;
                return old.Tile;
            }
            var tile = new TerrainTile { X = chosen.Item1, Z = chosen.Item2, WaterLevel = water, MeasuredAt = new double?[25] };
            var complete = true;
            for (var iz = 0; iz < 5; iz++)
                for (var ix = 0; ix < 5; ix++)
                {
                    var i = iz * 5 + ix;
                    var at = sample(tile.X * TileSize + ix * Spacing, tile.Z * TileSize + iz * Spacing);
                    Samples++;
                    if (at.HasValue && !double.IsNaN(at.Value.Height) && !double.IsInfinity(at.Value.Height))
                    {
                        tile.Heights[i] = Math.Round(at.Value.Height, 1);
                        tile.Biomes[i] = at.Value.Biome;
                        tile.MeasuredAt[i] = now;
                    }
                    else
                    {
                        complete = false;
                        // An unloaded heightmap does not erase earlier knowledge.
                        tile.Heights[i] = old?.Tile.Heights[i];
                        tile.Biomes[i] = old?.Tile.Biomes[i];
                        tile.MeasuredAt[i] = old?.Tile.MeasuredAt[i];
                    }
                }
            Builds++;
            var changed = old == null || !old.Sent || old.Tile.WaterLevel != water ||
                !old.Tile.Heights.SequenceEqual(tile.Heights) || !old.Tile.Biomes.SequenceEqual(tile.Biomes);
            entries[chosen] = new Entry { Tile = tile, Due = now + (complete ? 30 : 5), Sent = true };
            if (entries.Count > Capacity)
            {
                var farthest = entries.Keys.OrderByDescending(k =>
                    Math.Pow(k.Item1 - cx, 2) + Math.Pow(k.Item2 - cz, 2)).First();
                entries.Remove(farthest);
            }
            if (changed) Changes++;
            // Equal freshly measured values confirm freshness without extra
            // sampling. Completely unloaded unchanged tiles need no receipt.
            if (!changed && tile.MeasuredAt.SequenceEqual(old.Tile.MeasuredAt)) return null;
            return tile;
        }
    }
}
