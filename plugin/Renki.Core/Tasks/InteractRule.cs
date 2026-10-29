using Renki.Protocol;

namespace Renki.Tasks
{
    /// <summary>
    /// What interact may use, by the kind of thing it is (docs/BRIDGE-PROTOCOL.md
    /// 7.2 and section 8). The plugin names the kind from the game's type; this
    /// decides, so the rule is the same whatever the type is called.
    /// </summary>
    public static class InteractRule
    {
        public const string Boss = "boss", Ship = "ship", Pickable = "pickable", Drop = "drop",
            Container = "container", Station = "station", Text = "text", Ward = "ward", Other = "other";

        /// <summary>Null when interact may use a thing of this kind, or the
        /// refusal: a reason and what to say.</summary>
        public static Outcome Check(string kind, SafetyConfig safety)
        {
            switch (kind)
            {
                case Boss:
                    return safety.AllowBossSummoning ? null
                        : Outcome.Failed(Wire.ReasonNotAllowed, "summoning a boss needs AllowBossSummoning");
                case Ship:
                    return safety.AllowSailing ? null
                        : Outcome.Failed(Wire.ReasonNotAllowed, "taking the helm needs AllowSailing");
                case Pickable:
                    return safety.AllowGather ? null
                        : Outcome.Failed(Wire.ReasonNotAllowed, "picking needs AllowGather");
                // These have tasks of their own, or open a window the bot
                // cannot use and would leave open.
                case Drop:
                    return Outcome.Failed(Wire.ReasonInvalidTarget, "a drop is for pickup");
                case Container:
                    return Outcome.Failed(Wire.ReasonInvalidTarget, "a chest is for deposit and withdraw");
                case Station:
                    return Outcome.Failed(Wire.ReasonInvalidTarget, "a crafting station opens a window the bot cannot use");
                case Text:
                    return Outcome.Failed(Wire.ReasonInvalidTarget, "it asks for text, which the bot cannot type");
                // Using one turns it on or off, or adds the user to it: that
                // is its owner's to do.
                case Ward:
                    return Outcome.Failed(Wire.ReasonNotAllowed, "a ward is its owner's to set");
            }
            return null;
        }
    }
}
