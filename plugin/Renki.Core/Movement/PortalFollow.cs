using Renki.Protocol;
using Renki.Tasks;

namespace Renki.Movement
{
    /// <summary>An observed native connection, never inferred from matching tags.</summary>
    public sealed class FollowPortalLink
    {
        public readonly string Source, Destination;
        public readonly V3 Entry;
        public FollowPortalLink(string source, string destination, V3 entry)
        { Source = source; Destination = destination; Entry = entry; }
    }

    /// <summary>Short-lived departure evidence survives replacement of a follow by passive work.</summary>
    public sealed class PortalLeaderTrail
    {
        public const double RememberSeconds = 8, NearPortal = 2, RemoteJump = 32;
        FollowPortalLink near, departed;
        V3 last;
        double seen = double.NegativeInfinity, departedAt;

        public double Seen => seen;
        public void Observe(V3 position, FollowPortalLink link, double now)
        {
            if (now - seen <= RememberSeconds && near != null && (position - last).Length > RemoteJump)
            { departed = near; departedAt = seen; }
            near = link != null && (position - link.Entry).Length <= NearPortal ? link : null;
            last = position; seen = now;
        }

        public FollowPortalLink Departure(double now, bool visible, V3 position)
        {
            if (departed != null && now >= departedAt && now - departedAt <= RememberSeconds &&
                (!visible || (position - departed.Entry).Length > RemoteJump)) return departed;
            return !visible && now >= seen && now - seen <= RememberSeconds ? near : null;
        }
    }

    public enum PortalFollowAction { Walk, Hold, Arrived, Resume, Failed }

    /// <summary>Bounds entry, native teleport and arrival separately from the ordinary lost timer.</summary>
    public sealed class PortalFollowCrossing
    {
        public const double ApproachSeconds = 45, TriggerSeconds = 4, TeleportSeconds = 60, ArrivalSeconds = 15;
        enum Phase { Approach, Teleport, Arrival, End }
        readonly FollowPortalLink link;
        Phase phase;
        double since, entered = double.NaN;
        public Outcome Failure { get; private set; }
        public bool Arriving => phase == Phase.Teleport || phase == Phase.Arrival;
        public PortalFollowCrossing(FollowPortalLink link, double now) { this.link = link; since = now; }

        public bool Allows(string source, string destination) => phase == Phase.Approach && link != null &&
            link.Source == source && link.Destination == destination;

        public PortalFollowAction Tick(double now, bool teleporting, bool leaderHere, bool linkValid, bool cargoAllowed, bool inTrigger)
        {
            if (phase == Phase.End) return Failure == null ? PortalFollowAction.Resume : PortalFollowAction.Failed;
            if (teleporting)
            {
                if (phase != Phase.Teleport) { phase = Phase.Teleport; since = now; }
                return now - since > TeleportSeconds ? Fail("teleport did not finish") : PortalFollowAction.Hold;
            }
            if (phase == Phase.Teleport)
            { phase = Phase.Arrival; since = now; return PortalFollowAction.Arrived; }
            if (phase == Phase.Arrival)
            {
                if (leaderHere) { phase = Phase.End; return PortalFollowAction.Resume; }
                return now - since > ArrivalSeconds ? Fail("leader not found after portal arrival") : PortalFollowAction.Hold;
            }
            // A leader coming back on this side cancels the attempted crossing.
            if (leaderHere) { phase = Phase.End; return PortalFollowAction.Resume; }
            if (!linkValid) return Fail("portal connection changed or source unloaded");
            if (!cargoAllowed) return Fail("can't take my cargo through the portal", Wire.ReasonPortalCargo);
            if (now - since > ApproachSeconds) return Fail("could not reach the portal trigger");
            if (inTrigger)
            {
                if (double.IsNaN(entered)) entered = now;
                if (now - entered > TriggerSeconds) return Fail("the game refused portal entry");
                return PortalFollowAction.Hold;
            }
            return PortalFollowAction.Walk;
        }

        PortalFollowAction Fail(string detail, string reason = Wire.ReasonPortalFailed)
        { phase = Phase.End; Failure = Outcome.Failed(reason, detail); return PortalFollowAction.Failed; }
        public void Cancel() { phase = Phase.End; }
    }
}
