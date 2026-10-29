using System;
using System.Linq;
using System.Text;

namespace Renki.Tasks
{
    /// <summary>
    /// The plugin's own safety switches (docs/BRIDGE-PROTOCOL.md section 8).
    /// They come from the BepInEx config, and welcome.allowedTasks can only
    /// narrow them. stop is always permitted, because stopping is never the
    /// unsafe choice.
    /// </summary>
    public sealed class SafetyConfig
    {
        public bool Enabled = true;
        public bool AllowMovement = true;
        public bool AllowChat = true;
        public bool AllowGather;
        public bool AllowDrop = true;
        public bool AllowCombat;
        public bool AllowSailing;
        public bool AllowBossSummoning;
        public bool AllowRawControls;

        public bool Permits(string task)
        {
            if (!Enabled) return false;
            switch (task)
            {
                case "stop": return true;
                // Eating carried food does not move or gather.
                case "use_item": return true;
                case "move_to":
                case "follow":
                // It walks there; what it may use is InteractRule's call.
                case "interact":
                // Running away is never the unsafe choice, so it needs no
                // AllowCombat.
                case "flee":
                // The bot's own gear: changing it moves nothing, but it is
                // the body's to do.
                case "equip":
                case "unequip":
                case "look_at": return AllowMovement;
                case "say":
                case "emote": return AllowChat;
                // They walk to what they gather, so movement must be allowed too.
                case "pickup":
                case "deposit":
                case "withdraw":
                case "harvest": return AllowGather && AllowMovement;
                // Drops at the bot's feet, never what it has equipped.
                case "drop": return AllowDrop;
                case "attack":
                case "guard": return AllowCombat;
                case "controls": return AllowRawControls;
                default: return false;
            }
        }
    }

    /// <summary>
    /// Rate limit and clean-up for <c>say</c>: one line per 2 s with a burst
    /// of 3, no leading slash (so the bot can never type a console or chat
    /// command), and no longer than a chat line.
    /// </summary>
    public sealed class SayLimiter
    {
        public const int MaxLength = 200;

        readonly double interval;
        readonly int burst;
        double tokens;
        double last = double.NaN;

        public SayLimiter(double intervalS = 2, int burst = 3)
        {
            interval = intervalS;
            this.burst = burst;
            tokens = burst;
        }

        /// <summary>Takes a token at time now (seconds), or returns false when none is left.</summary>
        public bool TryTake(double now)
        {
            if (!double.IsNaN(last)) tokens = Math.Min(burst, tokens + (now - last) / interval);
            last = now;
            if (tokens < 1) return false;
            tokens -= 1;
            return true;
        }

        /// <summary>The text as it may be said: one line, no leading slash,
        /// at most MaxLength characters. Empty means nothing to say.</summary>
        public static string Clean(string text)
        {
            if (text == null) return "";
            var sb = new StringBuilder(text.Length);
            foreach (var c in text) sb.Append(char.IsControl(c) ? ' ' : c);
            var s = string.Join(" ", sb.ToString().Split((char[])null, StringSplitOptions.RemoveEmptyEntries));
            s = s.TrimStart('/', ' ');
            if (s.Length > MaxLength) s = s.Substring(0, MaxLength - 1) + "…";
            return s;
        }
    }
}
