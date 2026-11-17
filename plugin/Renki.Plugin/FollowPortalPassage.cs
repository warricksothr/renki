using System;
using System.Collections.Generic;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Native walking/trigger adapter. Only this crossing may admit its exact live link.</summary>
    sealed class FollowPortalPassage
    {
        static readonly int TriggerMask = LayerMask.GetMask("character_trigger", "Default", "Default_small", "piece", "static_solid");
        static readonly Collider[] nearby = new Collider[64];
        static FollowPortalPassage planning;
        readonly FollowPortalLink link;
        readonly bool ordered;
        readonly PortalFollowCrossing crossing;
        readonly Walker walker = new Walker();
        readonly TeleportWorld source;
        readonly Collider[] triggers;
        bool walking;
        public bool Done { get; private set; }

        public FollowPortalPassage(FollowPortalLink link, bool ordered = false)
        {
            this.link = link;
            this.ordered = ordered;
            crossing = new PortalFollowCrossing(link, Time.unscaledTime, ordered);
            source = link != null ? Resolve(link.Source) : null;
            triggers = source != null ? source.GetComponentsInChildren<Collider>() : Array.Empty<Collider>();
            Tracer.Emit("follow", link != null ? "portal departure" : "teleport observed", new Dictionary<string, object>
            { ["source"] = link?.Source, ["destination"] = link?.Destination, ["entry"] = link?.Entry.ToArray() });
        }

        internal static TeleportWorld Resolve(string id)
        {
            if (id == null) return null;
            var parts = id.Split(':');
            return parts.Length == 2 && long.TryParse(parts[0], out var user) && uint.TryParse(parts[1], out var serial)
                ? ZNetScene.instance?.FindInstance(new ZDOID(user, serial))?.GetComponent<TeleportWorld>() : null;
        }

        internal static string Destination(TeleportWorld portal)
        {
            var view = portal != null ? portal.GetComponent<ZNetView>() : null;
            var record = view != null && view.IsValid() ? view.GetZDO() : null;
            var target = record != null ? record.GetConnectionZDOID(ZDOExtraData.ConnectionType.Portal) : ZDOID.None;
            return target != ZDOID.None ? target.ToString() : null;
        }

        internal static FollowPortalLink Near(Vector3 position)
        {
            var count = Physics.OverlapSphereNonAlloc(position + Vector3.up, 3, nearby, TriggerMask, QueryTriggerInteraction.Collide);
            FollowPortalLink found = null;
            try
            {
                if (count == nearby.Length) return null;
                for (var i = 0; i < count; i++)
                {
                    var collider = nearby[i];
                    if (!NavigationObstacles.IsPortal(collider)) continue;
                    var portal = collider.GetComponentInParent<TeleportWorld>();
                    var destination = Destination(portal);
                    if (destination == null) continue;
                    var entry = collider.bounds.center;
                    entry.y = portal.transform.position.y;
                    if (Vector3.Distance(position, entry) > PortalLeaderTrail.NearPortal) continue;
                    var id = Perceiver.Id(portal.GetComponent<ZNetView>());
                    // Two nearby portals are ambiguous evidence. Do not guess.
                    if (found != null && found.Source != id) return null;
                    found = new FollowPortalLink(id, destination, Convert.ToV3(entry));
                }
                return found;
            }
            finally { Array.Clear(nearby, 0, nearby.Length); }
        }

        internal static bool MayEnter(Collider collider) => (planning ?? Driver.PortalPassage)?.Allows(collider) == true;
        bool Allows(Collider collider)
        {
            if (!NavigationObstacles.IsPortal(collider)) return false;
            var portal = collider.GetComponentInParent<TeleportWorld>();
            return portal != null && crossing.Allows(Perceiver.Id(portal.GetComponent<ZNetView>()), Destination(portal));
        }

        public Outcome Tick(Player player, Player leader)
        {
            var teleporting = player.IsTeleporting();
            var valid = source != null && source.isActiveAndEnabled && Destination(source) == link.Destination;
            var inTrigger = false;
            if (valid)
                foreach (var trigger in triggers)
                    if (NavigationObstacles.IsPortal(trigger) && trigger.enabled && trigger.gameObject.activeInHierarchy &&
                        (trigger.ClosestPoint(player.transform.position + Vector3.up) - (player.transform.position + Vector3.up)).sqrMagnitude < .0001f)
                    { inTrigger = true; break; }
            var leaderHere = leader != null && !leader.IsDead() && (crossing.Arriving || link == null
                ? Vector3.Distance(player.transform.position, leader.transform.position) < 128
                : Vector3.Distance(Convert.ToVector3(link.Entry), leader.transform.position) <= PortalLeaderTrail.RemoteJump);
            if (ordered)
            {
                var destination = Resolve(link.Destination);
                leaderHere = crossing.Arriving && destination != null && destination.isActiveAndEnabled &&
                    PortalArrival.Verified(Convert.ToV3(player.transform.position), Convert.ToV3(destination.transform.position),
                        player.IsOnGround(), player.IsSwimming(), teleporting);
                if (!crossing.Arriving && valid && !PrivateArea.CheckAccess(source.transform.position, 0f, false, false))
                { Cancel(); return Outcome.Failed(Wire.ReasonNotAllowed, "portal is behind a ward"); }
            }
            var action = crossing.Tick(Time.unscaledTime, teleporting, leaderHere, valid,
                !valid || player.IsTeleportable(source.m_allowAllItems), inTrigger);
            if (action != PortalFollowAction.Walk) Driver.Stand();
            switch (action)
            {
                case PortalFollowAction.Failed:
                    Done = true;
                    Tracer.Emit("follow", "portal failed", new Dictionary<string, object>
                    { ["source"] = link?.Source, ["reason"] = crossing.Failure.Reason, ["detail"] = crossing.Failure.Detail });
                    return crossing.Failure;
                case PortalFollowAction.Arrived:
                    WorldNavigation.Reset();
                    Players.Forget();
                    Tracer.Emit("follow", "portal arrived", new Dictionary<string, object> { ["pos"] = Convert.ToArray(player.transform.position) });
                    return null;
                case PortalFollowAction.Resume:
                    Done = true;
                    if (ordered)
                    {
                        var destination = Resolve(link.Destination);
                        var receipt = new Dictionary<string, object>
                        {
                            ["source"] = link.Source,
                            ["destination"] = link.Destination,
                            ["arrival"] = Convert.ToArray(player.transform.position),
                            ["observedReturnTo"] = Destination(destination)
                        };
                        Tracer.Emit("portal", "verified arrival", receipt);
                        return Outcome.Succeeded(Newtonsoft.Json.JsonConvert.SerializeObject(receipt));
                    }
                    return null;
                case PortalFollowAction.Hold:
                    return null;
            }
            // Ask the native portal to load its target before entering; it
            // only retries OnTriggerEnter when we actually cross the edge.
            if (!source.TargetFound() && Vector3.Distance(player.transform.position, Convert.ToVector3(link.Entry)) < 3)
            { Driver.Stand(); return null; }
            // Walker may stand while asynchronously validating a route. Keep
            // planning admission scoped to this call, independently of inputs.
            planning = this;
            try
            {
                if (!walking)
                {
                    walker.ResetStuck(player.transform.position);
                    walker.SetGoal(player.transform.position, Convert.ToVector3(link.Entry), within: .1, dryGround: true, avoidHostiles: ordered);
                    walking = true;
                }
                var failed = Walker.Stranded(player) ?? walker.Refused();
                if (failed != null) { Cancel(); return failed; }
                if (!walker.Steer(player.transform.position, false))
                { Cancel(); return Outcome.Failed(Wire.ReasonPortalFailed, "could not walk into the selected portal"); }
                Driver.PortalPassage = this;
                return null;
            }
            finally { planning = null; }
        }

        public void Cancel() { crossing.Cancel(); Done = true; Driver.Stand(); }
    }

    static partial class Players
    {
        sealed class Trail
        {
            internal string Name, PlayerId, UserId;
            internal readonly PortalLeaderTrail History = new PortalLeaderTrail();
        }
        static readonly Dictionary<string, Trail> trails = new Dictionary<string, Trail>();
        static readonly List<string> expired = new List<string>();
        static float nextObservation;

        internal static void Forget() { trails.Clear(); nextObservation = 0; }
        internal static void Observe()
        {
            var now = Time.unscaledTime;
            if (now < nextObservation) return;
            nextObservation = now + .1f;
            expired.Clear();
            foreach (var pair in trails)
                if (now - pair.Value.History.Seen > PortalLeaderTrail.RememberSeconds) expired.Add(pair.Key);
            foreach (var key in expired) trails.Remove(key);
            foreach (var player in Player.GetAllPlayers())
            {
                if (player == null || player == Player.m_localPlayer || player.m_nview == null || !player.m_nview.IsValid()) continue;
                var id = player.GetZDOID().ToString();
                if (player.IsDead()) { trails.Remove(id); continue; }
                if (!trails.TryGetValue(id, out var trail))
                {
                    if (trails.Count >= 64) continue;
                    trails[id] = trail = new Trail { Name = player.GetPlayerName(), PlayerId = player.GetPlayerID().ToString(), UserId = player.GetZDOID().UserID.ToString() };
                }
                var position = player.transform.position;
                trail.History.Observe(Convert.ToV3(position), FollowPortalPassage.Near(position), now);
            }
        }

        internal static FollowPortalLink Departure(string player, Player visible)
        {
            foreach (var trail in trails.Values)
                if (string.Equals(trail.Name, player, StringComparison.OrdinalIgnoreCase) || trail.PlayerId == player || trail.UserId == player)
                    return trail.History.Departure(Time.unscaledTime, visible != null,
                        visible != null ? Convert.ToV3(visible.transform.position) : V3.Zero);
            return null;
        }
    }
}
