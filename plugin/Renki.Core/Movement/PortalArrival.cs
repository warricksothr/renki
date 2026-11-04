namespace Renki.Movement
{
    /// <summary>Loaded exit proximity is verified after native teleport, including its floor.</summary>
    public static class PortalArrival
    {
        public static bool Verified(V3 player, V3 destination, bool grounded, bool swimming, bool teleporting) =>
            grounded && !swimming && !teleporting && System.Math.Abs(player.Y - destination.Y) <= 1.5 &&
            (player - destination).Length <= 4;
    }
}
