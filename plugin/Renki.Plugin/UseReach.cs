using System.Linq;
using Renki.Movement;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    static class UseReach
    {
        internal static bool Near(Player player, GameObject target, float reach)
        {
            var eye = player.m_eye != null ? player.m_eye.position : player.transform.position + Vector3.up * 1.6f;
            bool Sees(Vector3 at)
            {
                var delta = at - eye;
                if (delta.magnitude > reach) return false;
                if (delta.magnitude < .05f) return true;
                return InteractionSight.Visible(Physics.RaycastAll(eye, delta.normalized, reach, player.m_interactMask)
                    .Select(hit => new SightHit(hit.distance,
                        hit.collider.attachedRigidbody != null && hit.collider.attachedRigidbody.gameObject == player.gameObject,
                        hit.collider.transform.IsChildOf(target.transform))));
            }
            var any = false;
            foreach (var collider in target.GetComponentsInChildren<Collider>())
            {
                if (!collider.enabled || collider.isTrigger) continue;
                any = true;
                if (!(collider is MeshCollider mesh && !mesh.convex) && Sees(collider.ClosestPoint(eye))) return true;
                if (Sees(collider.bounds.center)) return true;
            }
            return !any && Arrival.Reached(Convert.ToV3(player.transform.position), Convert.ToV3(target.transform.position), reach);
        }
    }
}
