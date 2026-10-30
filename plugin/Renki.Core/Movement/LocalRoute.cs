using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    public enum RouteStatus { Searching, Found, NoRoute }

    /// <summary>Bounded A* around obstacles the navmesh missed. A probe
    /// returns a supported, clear destination or null. Heights are part of
    /// node identity, so a floor and the ground below it stay distinct.</summary>
    public sealed class LocalRoute
    {
        // Keep a cardinal step below the host's 0.6 m support-height bound.
        // A 0.75 m sample skips three 0.25 m stair treads on a 45-degree
        // staircase and falsely disconnects its upper and lower floors.
        public const double Cell = .5;
        public const int MaxNodes = 4096;
        public const double Range = 48;
        public const double FrontierDistance = 16;
        public const double FrontierProgress = 8;

        sealed class Node
        {
            public int X, Z;
            public V3 Pos;
            public double Cost, Score;
            public Node Parent;
            public long Serial;
        }

        readonly V3 start, goal;
        readonly double within;
        readonly double initialDistance;
        readonly Func<V3, V3, V3?> probe;
        readonly Dictionary<(int, int, int), double> costs = new Dictionary<(int, int, int), double>();
        readonly SortedSet<Node> open = new SortedSet<Node>(Comparer<Node>.Create((a, b) =>
        {
            var score = a.Score.CompareTo(b.Score);
            return score != 0 ? score : a.Serial.CompareTo(b.Serial);
        }));
        long serial;
        public RouteStatus Status { get; private set; } = RouteStatus.Searching;
        public IReadOnlyList<V3> Path { get; private set; }
        public int Expanded { get; private set; }
        public int Probes { get; private set; }
        /// <summary>A found path is only a checked leg, not arrival at the destination.</summary>
        public bool ProgressLeg { get; }

        public LocalRoute(V3 start, V3 goal, Func<V3, V3, V3?> probe, double within = .25)
        {
            this.start = start;
            this.goal = goal;
            this.probe = probe;
            this.within = within;
            initialDistance = V3.FlatDistance(start, goal);
            ProgressLeg = initialDistance > FrontierDistance * 1.5;
            Add(new Node { Pos = start });
        }

        public RouteStatus Advance(int budget = 8)
        {
            for (var i = 0; i < budget && Status == RouteStatus.Searching; i++)
            {
                if (open.Count == 0 || Expanded >= MaxNodes) { Status = RouteStatus.NoRoute; break; }
                var node = open.Min;
                open.Remove(node);
                if (costs[Key(node)] < node.Cost) { i--; continue; }
                Expanded++;
                if (Arrival.Reached(node.Pos, goal, within))
                {
                    Complete(node, node.Pos);
                    break;
                }
                // Plan only through loaded, probed ground. A distant goal's
                // height may belong to an unloaded hill or raised floor;
                // intermediate legs follow support, never that final height.
                if (ProgressLeg && V3.FlatDistance(start, node.Pos) >= FrontierDistance &&
                    initialDistance - V3.FlatDistance(node.Pos, goal) >= FrontierProgress)
                {
                    Complete(node, node.Pos);
                    break;
                }
                if (V3.FlatDistance(node.Pos, goal) <= within + Cell * 1.5 && Math.Abs(node.Pos.Y - goal.Y) <= Arrival.Level)
                {
                    Probes++;
                    // Grid cells can straddle the narrow space beside a
                    // solid goal. Probe a reachable arrival point, not the
                    // occupied centre of a chest/tree/other target.
                    var offset = (node.Pos - goal).Flat.Normalized * (within * .95);
                    var end = probe(node.Pos, new V3(goal.X + offset.X, node.Pos.Y, goal.Z + offset.Z));
                    if (end is V3 at && Arrival.Reached(at, goal, within))
                    {
                        Complete(node, at);
                        break;
                    }
                }
                for (var x = -1; x <= 1; x++)
                    for (var z = -1; z <= 1; z++)
                    {
                        if (x == 0 && z == 0) continue;
                        var nx = node.X + x;
                        var nz = node.Z + z;
                        var desired = new V3(start.X + nx * Cell, node.Pos.Y, start.Z + nz * Cell);
                        if (Math.Abs(desired.X - start.X) > Range || Math.Abs(desired.Z - start.Z) > Range) continue;
                        Probes++;
                        var point = probe(node.Pos, desired);
                        if (!(point is V3 pos)) continue;
                        var cost = node.Cost + (pos - node.Pos).Length;
                        var next = new Node { X = nx, Z = nz, Pos = pos, Cost = cost, Parent = node };
                        if (costs.TryGetValue(Key(next), out var previous) && previous <= cost + 1e-6) continue;
                        Add(next);
                    }
            }
            return Status;
        }

        static (int, int, int) Key(Node n) => (n.X, n.Z, (int)Math.Round(n.Pos.Y * 4));

        void Complete(Node node, V3 at)
        {
            var path = new List<V3> { at };
            for (var p = node; p != null; p = p.Parent) path.Add(p.Pos);
            path.Reverse();
            Path = path;
            Status = RouteStatus.Found;
        }

        void Add(Node node)
        {
            costs[Key(node)] = node.Cost;
            node.Score = node.Cost + V3.FlatDistance(node.Pos, goal);
            node.Serial = serial++;
            open.Add(node);
        }
    }
}
