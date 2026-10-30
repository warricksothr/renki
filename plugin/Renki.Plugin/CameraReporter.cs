using System;
using System.Collections;
using Renki.Perception;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    // Captures the actual rendered framebuffer, including the HUD. GPU resize
    // precedes CPU readback; nothing moves or changes the game's camera.
    sealed class CameraReporter
    {
        readonly CaptureSchedule schedule = new CaptureSchedule();
        ScreenshotConfig config = new ScreenshotConfig();
        int generation;
        public void Configure(ScreenshotConfig value) { config = value ?? new ScreenshotConfig(); generation++; }
        public void Update(RenkiPlugin plugin, bool active)
        {
            if (schedule.Begin(Time.unscaledTimeAsDouble, config.IntervalS, active && config.Enabled))
                plugin.StartCoroutine(Capture(plugin, generation));
        }
        IEnumerator Capture(RenkiPlugin plugin, int expected)
        {
            yield return new WaitForEndOfFrame();
            RenderTexture full = null, small = null;
            Texture2D pixels = null;
            var previous = RenderTexture.active;
            try
            {
                var player = Player.m_localPlayer;
                var camera = Camera.main;
                if (expected != generation || !config.Enabled || !plugin.CameraActive || player == null || camera == null ||
                    Screen.width < 1 || Screen.height < 1 || Screen.width > 3840 || Screen.height > 2160) yield break;
                var started = System.Diagnostics.Stopwatch.StartNew();
                var connection = plugin.CameraConnection;
                var world = ZNet.instance.GetWorldName();
                var uid = ZNet.instance.GetWorldUID().ToString(System.Globalization.CultureInfo.InvariantCulture);
                var width = Math.Min(Screen.width, Math.Max(320, Math.Min(1280, config.Width)));
                var scale = Math.Min((double)width / Screen.width, 720.0 / Screen.height);
                width = Math.Max(1, (int)(Screen.width * scale));
                var height = Math.Max(1, (int)(Screen.height * scale));
                var frame = new Screenshot
                {
                    World = world,
                    WorldUid = uid,
                    CapturedAt = DateTime.UtcNow,
                    Pos = Convert.ToArray(player.transform.position),
                    CameraPos = Convert.ToArray(camera.transform.position),
                    CameraAngles = Convert.ToArray(camera.transform.eulerAngles),
                    Task = plugin.CameraTask,
                    Width = width,
                    Height = height
                };
                full = RenderTexture.GetTemporary(Screen.width, Screen.height, 0, RenderTextureFormat.ARGB32);
                small = RenderTexture.GetTemporary(width, height, 0, RenderTextureFormat.ARGB32);
                ScreenCapture.CaptureScreenshotIntoRenderTexture(full);
                Graphics.Blit(full, small);
                RenderTexture.active = small;
                pixels = new Texture2D(width, height, TextureFormat.RGB24, false);
                pixels.ReadPixels(new Rect(0, 0, width, height), 0, 0, false);
                frame.Jpeg = ImageConversion.EncodeToJPG(pixels, Math.Max(30, Math.Min(90, config.Quality)));
                frame.CaptureMs = started.Elapsed.TotalMilliseconds;
                if (frame.Jpeg.Length <= 512 * 1024 && expected == generation && plugin.CameraActive)
                    plugin.SendScreenshot(frame, connection);
                else RenkiPlugin.Log?.LogDebug("camera: frame discarded after bounds/configuration change");
            }
            finally
            {
                RenderTexture.active = previous;
                if (pixels != null) UnityEngine.Object.Destroy(pixels);
                if (small != null) RenderTexture.ReleaseTemporary(small);
                if (full != null) RenderTexture.ReleaseTemporary(full);
                schedule.Complete();
            }
        }
    }
}
