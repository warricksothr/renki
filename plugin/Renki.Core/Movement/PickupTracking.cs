namespace Renki.Movement
{
    /// <summary>A dropped product can settle less than a metre from its spawn
    /// point, on the other side of a small obstruction. Follow that movement
    /// without rebuilding paths for centimetre-scale physics jitter.</summary>
    public static class PickupTracking
    {
        public static bool Retarget(V3 planned, V3 observed) => (observed - planned).Length > .25;

        // Arrival uses the actual item, never the old navigation endpoint.
        public static bool Reached(V3 feet, V3 observed, double within) => Arrival.Reached(feet, observed, within);
    }
}
