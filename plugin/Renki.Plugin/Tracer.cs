using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;

namespace Renki.Plugin
{
    /// <summary>
    /// The plugin's own decisions: a change of gait, a new path, a stuck
    /// recovery. Each goes to the BepInEx log and, as a <c>trace</c> message,
    /// to the controller's audit log, stamped with the running task.
    /// </summary>
    static class Tracer
    {
        /// <summary>Where traces go besides the BepInEx log; set by the plugin.</summary>
        internal static Action<Trace> Sink;

        public static void Emit(string source, string msg, Dictionary<string, object> data = null)
        {
            var detail = data == null || data.Count == 0
                ? ""
                : " (" + string.Join(", ", data.Select(kv => $"{kv.Key} {Format(kv.Value)}")) + ")";
            RenkiPlugin.Log?.LogInfo($"{source}: {msg}{detail}");
            Sink?.Invoke(new Trace { Source = source, Msg = msg, Data = data });
        }

        static string Format(object v)
        {
            switch (v)
            {
                case double[] a: return "(" + string.Join(", ", a.Select(x => x.ToString("0.##"))) + ")";
                case double d: return d.ToString("0.##");
                case float f: return f.ToString("0.##");
                default: return v?.ToString() ?? "null";
            }
        }
    }
}
