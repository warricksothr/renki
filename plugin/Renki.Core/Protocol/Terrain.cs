using System.Collections.Generic;
using Newtonsoft.Json;

namespace Renki.Protocol
{
    // Fixed grid: 16 m tiles, 4 m spacing, 5x5 corner samples, rows +z.
    // Null heights/biomes mean no loaded terrain measurement.
    public sealed class Terrain : Message
    {
        public override string MessageType => Wire.Terrain;
        [JsonProperty("world")] public string World = "";
        [JsonProperty("scope", NullValueHandling = NullValueHandling.Ignore)] public TerrainScope Scope;
        [JsonProperty("capturedAt", DefaultValueHandling = DefaultValueHandling.Ignore)] public double CapturedAt;
        [JsonProperty("tiles")] public List<TerrainTile> Tiles = new List<TerrainTile>();
    }

    public sealed class TerrainScope
    {
        [JsonProperty("namespace")] public string Namespace;
        [JsonProperty("worldUid")] public string WorldUid;
        [JsonProperty("epoch")] public string Epoch;
        [JsonProperty("source")] public string Source;
        [JsonProperty("stream")] public string Stream;
        [JsonProperty("game")] public string Game;
        [JsonProperty("generator")] public int Generator;
        [JsonProperty("mods")] public string Mods;
    }

    public sealed class TerrainTile
    {
        [JsonProperty("x")] public int X;
        [JsonProperty("z")] public int Z;
        [JsonProperty("waterLevel")] public double WaterLevel;
        [JsonProperty("heights")] public double?[] Heights = new double?[25];
        [JsonProperty("biomes")] public string[] Biomes = new string[25];
        // Producer monotonic seconds. Carried values retain their original
        // timestamp; null means no measurement. Resends never renew it.
        [JsonProperty("measuredAt", NullValueHandling = NullValueHandling.Ignore)] public double?[] MeasuredAt;
    }

    public sealed class Location
    {
        [JsonProperty("groundHeight", NullValueHandling = NullValueHandling.Ignore)] public double? GroundHeight;
        [JsonProperty("support")] public string Support = "unknown";
        [JsonProperty("inWater")] public bool InWater;
        // Water above the character's feet, not seabed depth.
        [JsonProperty("waterDepth")] public double WaterDepth;
    }
}
