using System.Collections.Generic;
using Newtonsoft.Json;

namespace Renki.Protocol
{
    public sealed class ContainerSnapshot : Message
    {
        public override string MessageType => Wire.Container;
        [JsonProperty("entity")] public string Entity;
        [JsonProperty("pos")] public double[] Pos;
        [JsonProperty("size")] public int[] Size;
        [JsonProperty("items")] public List<InventoryItem> Items;
        [JsonProperty("task", NullValueHandling = NullValueHandling.Ignore)] public string Task;
        [JsonProperty("direction", NullValueHandling = NullValueHandling.Ignore)] public string Direction;
        [JsonProperty("moved", NullValueHandling = NullValueHandling.Ignore)] public Dictionary<string, int> Moved;
        [JsonProperty("remaining", NullValueHandling = NullValueHandling.Ignore)] public Dictionary<string, int> Remaining;
    }
}
