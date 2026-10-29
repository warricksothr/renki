using System;
using System.Collections.Generic;
using System.Linq;

namespace Renki.Movement
{
    /// <summary>Grounded positions actually visited by one movement task.
    /// An oversized trail is discarded, never silently truncated.</summary>
    public sealed class RouteTrail
    {
        public const int MaxPoints = 256;
        readonly List<V3> points = new List<V3>();
        public IReadOnlyList<V3> Points => points;
        public bool Overflowed { get; private set; }

        public void Reset(V3 start) { points.Clear(); Overflowed = false; points.Add(start); }

        public void Record(V3 pos, bool final = false)
        {
            if (Overflowed) return;
            if (points.Count > 0 && (points[points.Count - 1] - pos).Length < (final ? .01 : .5)) return;
            // Remove a return to recently visited ground, including a failed
            // approach before the successful entrance was found.
            for (var i = points.Count - 2; i >= 0; i--)
                if ((points[i] - pos).Length < .5)
                {
                    points.RemoveRange(i + 1, points.Count - i - 1);
                    break;
                }
            if (points.Count == MaxPoints) { points.Clear(); Overflowed = true; return; }
            points.Add(pos);
        }
    }

    public sealed class RouteHint
    {
        public long ID { get; }
        public IReadOnlyList<V3> Points { get; }
        public RouteHint(long id, IReadOnlyList<V3> points) { ID = id; Points = points; }
    }

    /// <summary>Hints only: every join and leg still needs live validation.</summary>
    public sealed class RouteMemory
    {
        public const int Capacity = 24;
        readonly List<RouteHint> routes = new List<RouteHint>();
        long serial;
        public int Count => routes.Count;
        public void Clear() => routes.Clear();
        public void Remove(long id) => routes.RemoveAll(r => r.ID == id);

        public long? Remember(RouteTrail trail)
        {
            if (trail.Overflowed || trail.Points.Count < 3 ||
                (trail.Points[0] - trail.Points[trail.Points.Count - 1]).Length < 4) return null;
            if (routes.Count == Capacity) routes.RemoveAt(0);
            routes.Add(new RouteHint(++serial, trail.Points.ToArray()));
            return serial;
        }

        public RouteHint Find(V3 from, V3 goal)
        {
            RouteHint best = null;
            var bestCost = double.PositiveInfinity;
            foreach (var route in routes)
            {
                var join = -1; var finish = -1;
                var startGap = 3.0; var endGap = 4.0;
                for (var i = 0; i < route.Points.Count; i++)
                {
                    var p = route.Points[i];
                    var gap = V3.FlatDistance(from, p);
                    if (Math.Abs(from.Y - p.Y) <= .65 && gap <= startGap) { join = i; startGap = gap; }
                    gap = V3.FlatDistance(goal, p);
                    if (Math.Abs(goal.Y - p.Y) <= Arrival.Level && gap <= endGap) { finish = i; endGap = gap; }
                }
                if (join < 0 || finish < 0 || join == finish) continue;
                var points = new List<V3>();
                var previous = from;
                var cost = 0.0;
                var step = join < finish ? 1 : -1;
                for (var i = join; ; i += step)
                {
                    var p = route.Points[i];
                    cost += (p - previous).Length;
                    points.Add(p); previous = p;
                    if (i == finish) break;
                }
                cost += endGap;
                if (cost > bestCost) continue;
                // Home may have been set at eye height. Probe the goal on
                // the recorded destination floor, then check actual arrival.
                points.Add(new V3(goal.X, previous.Y, goal.Z));
                best = new RouteHint(route.ID, points);
                bestCost = cost;
            }
            return best;
        }
    }

    /// <summary>Incremental live support/clearance checks of a recorded hint.
    /// Outputs short, supported legs for continued checks while walking.</summary>
    public sealed class RouteValidation
    {
        public const int MaxSamples = 512;
        readonly IReadOnlyList<V3> hint;
        readonly Func<V3, V3, V3?> probe;
        readonly V3 goal;
        readonly double within;
        readonly List<V3> path = new List<V3>();
        V3 from, legStart;
        int index, part, parts;
        public int Samples { get; private set; }
        public V3? Checking { get; private set; }
        public RouteStatus Status { get; private set; } = RouteStatus.Searching;
        public IReadOnlyList<V3> Path => path;

        public RouteValidation(V3 start, V3 goal, RouteHint hint, Func<V3, V3, V3?> probe, double within = .25)
        {
            from = legStart = start; this.goal = goal; this.hint = hint.Points; this.probe = probe;
            this.within = within;
        }

        public RouteStatus Advance(int budget = 8)
        {
            for (var n = 0; n < budget && Status == RouteStatus.Searching; n++)
            {
                if (path.Count > 0 && Arrival.Reached(from, goal, within))
                {
                    Status = RouteStatus.Found; break;
                }
                if (index == hint.Count)
                {
                    Status = Arrival.Reached(from, goal, within) ? RouteStatus.Found : RouteStatus.NoRoute;
                    break;
                }
                if (Samples >= MaxSamples) { Status = RouteStatus.NoRoute; break; }
                if (parts == 0) parts = Math.Max(1, (int)Math.Ceiling((hint[index] - legStart).Length / .5));
                var expected = legStart + (hint[index] - legStart) * (++part / (double)parts);
                Checking = expected;
                Samples++;
                var supported = probe(from, expected);
                if (!(supported is V3 at) || V3.FlatDistance(at, expected) > .1 || Math.Abs(at.Y - expected.Y) > .65)
                {
                    Status = RouteStatus.NoRoute; break;
                }
                from = at;
                if (path.Count == 0 || (path[path.Count - 1] - at).Length >= .01) path.Add(at);
                if (part == parts) { index++; part = parts = 0; legStart = from; }
            }
            return Status;
        }
    }
}
