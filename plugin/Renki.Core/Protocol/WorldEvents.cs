using System;
using Newtonsoft.Json;

namespace Renki.Protocol
{
    /// <summary>Null means unavailable; an empty sample is observed clear.</summary>
    public sealed class WorldEventStatus
    {
        [JsonProperty("randomEvent", NullValueHandling = NullValueHandling.Ignore)] public AreaEvent RandomEvent;
        [JsonProperty("localEvent", NullValueHandling = NullValueHandling.Ignore)] public EventNotice LocalEvent;
    }

    public sealed class EventNotice
    {
        [JsonProperty("name")] public string Name = "";
        [JsonProperty("message")] public string Message = "";
        [JsonProperty("kind")] public string Kind = "";
    }

    public sealed class AreaEvent
    {
        [JsonProperty("name")] public string Name = "";
        [JsonProperty("message")] public string Message = "";
        [JsonProperty("center")] public double[] Center = new double[3];
        [JsonProperty("radius")] public double Radius;
        [JsonProperty("elapsedS")] public double ElapsedS;
        [JsonProperty("durationS")] public double DurationS;
        [JsonProperty("timerMayPause")] public bool TimerMayPause;
        [JsonProperty("inside")] public bool Inside;

        static bool Finite(double x) => !double.IsNaN(x) && !double.IsInfinity(x);
        static bool Position(double[] p) => p != null && p.Length == 3 && Finite(p[0]) && Finite(p[1]) && Finite(p[2]);
        public bool Valid() => !string.IsNullOrEmpty(Name) && Position(Center) && Finite(Radius) && Radius > 0 && Finite(ElapsedS) && ElapsedS >= 0 && Finite(DurationS);

        public bool Contains(double[] p)
        {
            if (!Valid() || !Position(p) || p[1] > 3000) return false;
            var x = p[0] - Center[0];
            var z = p[2] - Center[2];
            return Math.Sqrt(x * x + z * z) < Radius;
        }
    }
}
