using System;
using Newtonsoft.Json;

namespace Renki.Protocol
{
    public sealed class CaptureProof
    {
        [JsonProperty("token")] public string Token;
        [JsonProperty("at")] public DateTime At;
    }
}
