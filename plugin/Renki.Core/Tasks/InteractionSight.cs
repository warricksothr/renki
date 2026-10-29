using System.Collections.Generic;

namespace Renki.Tasks
{
    public readonly struct SightHit
    {
        public readonly double Distance;
        public readonly bool Self, Target;

        public SightHit(double distance, bool self, bool target)
        {
            Distance = distance;
            Self = self;
            Target = target;
        }
    }

    /// <summary>The nearest hit other than the player decides visibility.
    /// A player's own body can cross the ray from their eye, as in the
    /// game's hover check; walls and floors must still stop it.</summary>
    public static class InteractionSight
    {
        public static bool Visible(IEnumerable<SightHit> hits)
        {
            var nearest = double.PositiveInfinity;
            var visible = false;
            foreach (var hit in hits)
            {
                if (hit.Self) continue;
                if (hit.Distance < nearest)
                {
                    nearest = hit.Distance;
                    visible = hit.Target;
                }
                else if (hit.Distance == nearest && !hit.Target) visible = false;
            }
            return visible;
        }
    }
}
