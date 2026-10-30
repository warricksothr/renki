using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Globalization;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using HarmonyLib;
using Renki.Perception;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Loaded heightmaps only, without collision casts or generating
    /// unexplored world terrain. One tile per update; one changed batch/s.</summary>
    sealed class TerrainReporter
    {
        TerrainCache cache = new TerrainCache();
        string world;
        TerrainScope scope;
        public string Namespace = "local", Epoch = "1", Source;
        readonly Dictionary<(int, int), TerrainTile> pending = new Dictionary<(int, int), TerrainTile>();
        readonly Stopwatch watch = new Stopwatch();
        double totalMs, worstMs;
        float nextSend, nextReport;
        public static TerrainReporter Current;

        public void Resend() { cache.Resend(); }

        public void Update(RenkiPlugin plugin, Player p, double radius)
        {
            var net = ZNet.instance;
            var current = net != null ? net.GetWorldName() : "";
            if (string.IsNullOrEmpty(current)) return;
            var uid = net.GetWorldUID().ToString(CultureInfo.InvariantCulture);
            if (current != world || uid != scope?.WorldUid)
            {
                cache = new TerrainCache(); pending.Clear(); world = current;
                var mods = string.Join("\n", BepInEx.Bootstrap.Chainloader.PluginInfos.Values
                    .Select(info => info.Metadata.GUID + "@" + info.Metadata.Version).OrderBy(v => v, StringComparer.Ordinal));
                string fingerprint;
                using (var hash = SHA256.Create())
                    fingerprint = BitConverter.ToString(hash.ComputeHash(Encoding.UTF8.GetBytes(mods))).Replace("-", "").ToLowerInvariant();
                scope = new TerrainScope
                {
                    Namespace = Namespace,
                    Epoch = Epoch,
                    Source = Source,
                    WorldUid = uid,
                    Stream = Guid.NewGuid().ToString("N"),
                    Game = global::Version.GetVersionString(),
                    Generator = net.GetWorld().m_worldGenVersion,
                    Mods = fingerprint
                };
            }
            var at = p.transform.position;
            var forward = p.GetVelocity();
            if (forward.sqrMagnitude < .1f) forward = p.transform.forward;
            forward.y = 0; forward.Normalize();
            watch.Restart();
            var tile = cache.Step(at.x, at.z, forward.x, forward.z, Time.unscaledTimeAsDouble,
                ZoneSystem.instance != null ? ZoneSystem.instance.m_waterLevel : 30,
                Sample, (int)Math.Ceiling(Math.Min(64, radius) / TerrainCache.TileSize));
            watch.Stop();
            totalMs += watch.Elapsed.TotalMilliseconds;
            worstMs = Math.Max(worstMs, watch.Elapsed.TotalMilliseconds);
            if (tile != null) pending[(tile.X, tile.Z)] = tile;
            if (Time.unscaledTime >= nextSend)
            {
                nextSend = Time.unscaledTime + 1;
                if (pending.Count > 0)
                {
                    plugin.Send(new Terrain
                    {
                        World = world,
                        Scope = scope,
                        CapturedAt = Time.unscaledTimeAsDouble,
                        Tiles = new List<TerrainTile>(pending.Values)
                    });
                    pending.Clear();
                }
            }
            if (Time.unscaledTime < nextReport) return;
            nextReport = Time.unscaledTime + 30;
            Tracer.Emit("terrain", "cache", new Dictionary<string, object>
            {
                ["tracked"] = cache.Count,
                ["builds"] = cache.Builds,
                ["samples"] = cache.Samples,
                ["changes"] = cache.Changes,
                ["ms"] = Math.Round(totalMs, 2),
                ["worstFrameMs"] = Math.Round(worstMs, 3),
            });
            totalMs = worstMs = 0;
        }

        static TerrainSample? Sample(double x, double z)
        {
            var at = new Vector3((float)x, 0, (float)z);
            var hm = Heightmap.FindHeightmap(at);
            if (hm == null || !hm.GetWorldHeight(at, out var h)) return null;
            return new TerrainSample(h, hm.GetBiome(at).ToString());
        }

        [HarmonyPatch(typeof(Heightmap), nameof(Heightmap.Regenerate))]
        static class Changed
        {
            static void Postfix(Heightmap __instance)
            {
                var r = Current;
                if (r == null) return;
                var at = __instance.transform.position;
                var half = __instance.m_width * __instance.m_scale / 2;
                r.cache.Invalidate(at.x - half, at.z - half, at.x + half, at.z + half);
            }
        }
    }
}
