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
        public const double Cell = .75;
        public const int MaxNodes = 4096;
        public const double Range = 48;

        sealed class Node
        {
            public int X, Z;
            public V3 Pos;
            public double Cost, Score;
            public Node Parent;
            public long Serial;
        }

        readonly V3 start, goal;
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

        public LocalRoute(V3 start, V3 goal, Func<V3, V3, V3?> probe)
        {
            this.start = start;
            this.goal = goal;
            this.probe = probe;
            Add(new Node { Pos = start });
            if (Math.Abs(goal.X - start.X) > Range || Math.Abs(goal.Z - start.Z) > Range)
                Status = RouteStatus.NoRoute;
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
                if (V3.FlatDistance(node.Pos, goal) <= Cell * 1.5 && Math.Abs(node.Pos.Y - goal.Y) <= Arrival.Level)
                {
                    Probes++;
                    var end = probe(node.Pos, new V3(goal.X, node.Pos.Y, goal.Z));
                    if (end is V3 at && Arrival.Reached(at, goal, .25))
                    {
                        var path = new List<V3> { at };
                        for (var p = node; p != null; p = p.Parent) path.Add(p.Pos);
                        path.Reverse();
                        Path = path;
                        Status = RouteStatus.Found;
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

        void Add(Node node)
        {
            costs[Key(node)] = node.Cost;
            node.Score = node.Cost + V3.FlatDistance(node.Pos, goal);
            node.Serial = serial++;
            open.Add(node);
        }
    }
}
