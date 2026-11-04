using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Explicit normal-controls crossing, never a portal-tag interaction.</summary>
    sealed class PortalTravelTask : IBodyTask
    {
        readonly FollowPortalPassage passage;

        public PortalTravelTask(InteractArgs args)
        {
            var player = Player.m_localPlayer;
            var source = FollowPortalPassage.Resolve(args.Entity);
            if (source == null || !source.isActiveAndEnabled ||
                Vector3.Distance(player.transform.position, source.transform.position) > 16)
                throw new TaskRefused(Wire.ReasonInvalidTarget, "portal source must be loaded and within 16 m");
            if (player.IsTeleporting())
                throw new TaskRefused(Wire.ReasonPortalFailed, "already teleporting; no new crossing admitted");
            if (FollowPortalPassage.Destination(source) != args.PortalTo)
                throw new TaskRefused(Wire.ReasonPortalFailed, "portal connection differs from the ordered destination");
            var entry = source.transform.position;
            Collider trigger = null;
            foreach (var collider in source.GetComponentsInChildren<Collider>())
                if (NavigationObstacles.IsPortal(collider) && collider.enabled && collider.gameObject.activeInHierarchy)
                { trigger = collider; break; }
            if (trigger == null)
                throw new TaskRefused(Wire.ReasonPortalFailed, "portal entry trigger unavailable");
            entry = trigger.bounds.center;
            entry.y = source.transform.position.y;
            passage = new FollowPortalPassage(new FollowPortalLink(args.Entity, args.PortalTo, Convert.ToV3(entry)), ordered: true);
        }

        public Outcome Tick(double dt)
        {
            var player = Player.m_localPlayer;
            if (player == null || player.IsDead())
            { Cancel(); return Outcome.Failed(Wire.ReasonStopped, "not in the world"); }
            return passage.Tick(player, null);
        }

        public void Cancel() => passage.Cancel();
    }
}
