using HarmonyLib;

namespace Renki.Plugin
{
    /// <summary>
    /// Valheim joins on launch by itself: FejdStartup.ParseArguments, run
    /// from FejdStartup.Awake, handles `-joinserverwithcharacter host:port
    /// character x` and `-password`. The launch script relies on that. But
    /// Awake runs again whenever the menu scene loads, so after a logout the
    /// same arguments would rejoin at once. Once the controller has asked for
    /// a logout, they are ignored for the rest of this run. A rejoin after a
    /// kick or a server restart still happens, which a bot wants.
    /// </summary>
    [HarmonyPatch(typeof(FejdStartup), "ParseArguments")]
    static class AutoJoinPatch
    {
        public static bool Suppress;

        static bool Prefix() => !Suppress;
    }
}
