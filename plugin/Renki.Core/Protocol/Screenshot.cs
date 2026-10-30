using System;
using Newtonsoft.Json;

namespace Renki.Protocol
{
    public sealed class ScreenshotConfig
    {
        [JsonProperty("enabled")] public bool Enabled;
        [JsonProperty("intervalS")] public double IntervalS = 5;
        [JsonProperty("width")] public int Width = 960;
        [JsonProperty("quality")] public int Quality = 70;
    }

    public sealed class Screenshot : Message
    {
        public override string MessageType => Wire.Screenshot;
        [JsonProperty("world")] public string World;
        [JsonProperty("worldUid")] public string WorldUid;
        [JsonProperty("capturedAt")] public DateTime CapturedAt;
        [JsonProperty("pos")] public double[] Pos;
        [JsonProperty("cameraPos")] public double[] CameraPos;
        [JsonProperty("cameraAngles")] public double[] CameraAngles;
        [JsonProperty("task", NullValueHandling = NullValueHandling.Ignore)] public string Task;
        [JsonProperty("width")] public int Width;
        [JsonProperty("height")] public int Height;
        [JsonProperty("captureMs")] public double CaptureMs;
        [JsonProperty("jpeg")] public byte[] Jpeg;
    }
}
