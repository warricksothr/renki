using HarmonyLib;

namespace Renki.Plugin
{
    [HarmonyPatch(typeof(Raven), "Spawn")]
    static class TutorialHints
    {
        static bool Prefix(Raven.RavenText text) =>
            !(RenkiPlugin.Instance?.SuppressTutorials == true && text != null && text.m_isTutorial);
    }
}
