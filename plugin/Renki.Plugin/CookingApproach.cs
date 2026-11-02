using UnityEngine;

namespace Renki.Plugin
{
    // Cooking adds fuel reach, heat and fire clearance to the shared approach.
    sealed class CookingApproach
    {
        readonly InteractionApproach approach;

        internal CookingApproach(string id, CookingStation station, CookingFire fire)
        {
            approach = new InteractionApproach("cooking", "cooking", id, station.gameObject, (me, feet) =>
            {
                var reach = CookingTask.InteractionRange(me);
                var fuel = fire.PreparationTarget();
                return UseReach.From(me, station.gameObject, reach, feet) &&
                    (fuel == null || UseReach.From(me, fuel, reach, feet)) && fire.SafeFooting(feet) &&
                    EffectArea.IsPointInsideArea(feet + Vector3.up * .5f, EffectArea.Type.Burning, .4f) == null;
            });
        }

        internal Renki.Tasks.Outcome Tick(double dt, out bool waiting) => approach.Tick(dt, out waiting);
        internal void Cancel() => approach.Cancel();
    }
}
