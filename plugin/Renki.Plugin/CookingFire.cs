using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Matches the station's actual fire-check points to the supporting
    /// fire's burning-area bounds, including when those areas are inactive.</summary>
    sealed class CookingFire
    {
        static readonly int SmokeMask = LayerMask.GetMask("smoke");
        readonly CookingStation station;
        readonly string taskId;
        Fireplace fire;
        CookingFuel receipt;
        float began;
        public CookingFire(CookingStation station, string taskId) { this.station = station; this.taskId = taskId; }
        static int Wood(Player me) => me.GetInventory().GetAllItems().Where(i => i.m_dropPrefab?.name == "Wood").Sum(i => i.m_stack);

        List<Fireplace> Nearby()
        {
            var pieces = new List<Piece>(); Piece.GetAllPiecesInRadius(station.transform.position, 3, pieces);
            return pieces.Select(p => p.GetComponent<Fireplace>()).Where(f => f != null).Distinct().ToList();
        }

        internal GameObject PreparationTarget()
        {
            if (CookingTask.Heated(station)) return null;
            var candidates = Nearby().Where(f => Supports(f, station)).ToArray();
            return candidates.Length == 1 ? candidates[0].gameObject : null;
        }

        internal bool SafeFooting(Vector3 feet)
        {
            // Match the game's head-height smoke sphere at the proposed
            // standing position, rather than merely checking flame bounds.
            var player = Player.m_localPlayer;
            if (player == null || Physics.CheckSphere(feet + player.GetTopPoint() - player.transform.position + Vector3.up * .1f,
                .5f, SmokeMask, QueryTriggerInteraction.Collide)) return false;
            foreach (var nearby in Nearby())
                foreach (var original in BurningBounds(nearby))
                {
                    var bounds = original; bounds.Expand(.7f);
                    if (bounds.Contains(feet + Vector3.up * .3f) || bounds.Contains(feet + Vector3.up)) return false;
                }
            return true;
        }

        public Outcome Tick(Player me, out bool waiting)
        {
            waiting = false;
            if (receipt == null && CookingTask.Heated(station)) return null;
            waiting = true; Driver.Stand();
            if (receipt == null)
            {
                if (Wood(me) < 1) return Outcome.Failed(Wire.ReasonInvalidTarget, "carry one Wood to fuel the cooking fire");
                var candidates = Nearby().Where(f => Supports(f, station)).ToArray();
                if (candidates.Length != 1)
                    return Outcome.Failed(Wire.ReasonRefused, "no unique empty wood fire supports the cooking check points");
                fire = candidates[0];
                if (!PrivateArea.CheckAccess(fire.transform.position, 0f, false, false))
                    return Outcome.Failed(Wire.ReasonNotAllowed, "the supporting cooking fire is behind a ward");
                if (!UseReach.Near(me, fire.gameObject, CookingTask.InteractionRange(me)))
                    return Outcome.Failed(Wire.ReasonUnreachable, "the supporting fire is not in sight/reach");
                receipt = new CookingFuel(Wood(me), fire.m_nview.GetZDO().GetFloat(ZDOVars.s_fuel));
                began = Time.unscaledTime;
            }
            if (fire == null || fire.m_nview == null || !fire.m_nview.IsValid() ||
                !PrivateArea.CheckAccess(fire.transform.position, 0f, false, false))
                return Outcome.Failed(Wire.ReasonRefused, "the supporting fire or its access changed");
            var fuel = fire.m_nview.GetZDO().GetFloat(ZDOVars.s_fuel);
            var step = receipt.Next(Time.unscaledTime - began, Wood(me), fuel, CookingTask.Heated(station));
            if (step == CookingFuelAction.Failed) return Outcome.Failed(Wire.ReasonRefused, receipt.Failure);
            if (step == CookingFuelAction.Add)
            {
                var wood = me.GetInventory().GetAllItems().FirstOrDefault(i => i.m_dropPrefab?.name == "Wood");
                if (wood == null || !fire.UseItem(me, wood)) return Outcome.Failed(Wire.ReasonRefused, "the fire refused the wood");
                Tracer.Emit("cooking", "one supporting-fire fuel use requested", new Dictionary<string, object>
                { ["id"] = taskId, ["fire"] = Perceiver.Id(fire.m_nview) });
            }
            if (step == CookingFuelAction.Ready)
            {
                waiting = false;
                Tracer.Emit("cooking", "verified wood fuel and station heat receipt", new Dictionary<string, object>
                { ["id"] = taskId, ["fire"] = Perceiver.Id(fire.m_nview), ["fuel"] = fuel, ["woodConsumed"] = 1 });
            }
            return null;
        }

        static bool Supports(Fireplace fire, CookingStation station)
        {
            if (fire == null || !fire.m_canRefill || fire.m_infiniteFuel || fire.m_fuelItem?.name != "Wood" ||
                fire.m_nview == null || !fire.m_nview.IsValid() || fire.m_nview.GetZDO().GetFloat(ZDOVars.s_fuel) != 0 ||
                station.m_fireCheckRadius != .25f) return false;
            var points = station.m_fireCheckPoints != null && station.m_fireCheckPoints.Length != 0
                ? station.m_fireCheckPoints.Select(t => t.position).ToArray() : new[] { station.transform.position };
            var areas = BurningBounds(fire);
            return points.All(point => areas.Any(area => area.Contains(point)));
        }

        static List<Bounds> BurningBounds(Fireplace fire)
        {
            var areas = new List<Bounds>();
            foreach (var area in fire.GetComponentsInChildren<EffectArea>(true))
            {
                if ((area.m_type & EffectArea.Type.Burning) == 0) continue;
                var col = area.GetComponent<Collider>();
                Bounds bounds;
                if (col is BoxCollider box)
                {
                    bounds = new Bounds(box.transform.TransformPoint(box.center), Vector3.zero);
                    for (var i = 0; i < 8; i++) bounds.Encapsulate(box.transform.TransformPoint(box.center + Vector3.Scale(box.size * .5f,
                        new Vector3((i & 1) == 0 ? -1 : 1, (i & 2) == 0 ? -1 : 1, (i & 4) == 0 ? -1 : 1))));
                }
                else if (col is SphereCollider sphere)
                {
                    var scale = sphere.transform.lossyScale;
                    var radius = sphere.radius * Mathf.Max(Mathf.Abs(scale.x), Mathf.Abs(scale.y), Mathf.Abs(scale.z));
                    bounds = new Bounds(sphere.transform.TransformPoint(sphere.center), Vector3.one * radius * 2);
                }
                else continue;
                bounds.Expand(.25f); areas.Add(bounds);
            }
            return areas;
        }
    }
}
