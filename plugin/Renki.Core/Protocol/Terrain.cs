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
        [JsonProperty("tiles")] public List<TerrainTile> Tiles = new List<TerrainTile>();
    }

    public sealed class TerrainTile
    {
        [JsonProperty("x")] public int X;
        [JsonProperty("z")] public int Z;
        [JsonProperty("waterLevel")] public double WaterLevel;
        [JsonProperty("heights")] public double?[] Heights = new double?[25];
        [JsonProperty("biomes")] public string[] Biomes = new string[25];
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
