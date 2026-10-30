using Renki.Protocol;

namespace Renki.Perception
{
    public static class SessionPhase
    {
        public static string Resolve(bool loggingOut, bool hasPlayer, bool hasNetwork, bool dead)
        {
            if (!hasPlayer && !hasNetwork) return Wire.PhaseMenu;
            if (loggingOut) return Wire.PhaseLoggingOut;
            if (!hasPlayer) return Wire.PhaseConnecting;
            return dead ? Wire.PhaseDead : Wire.PhaseInWorld;
        }
    }
}
