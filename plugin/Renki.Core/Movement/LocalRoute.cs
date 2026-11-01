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
            public double Cost, Score, Remaining;
            public Node Parent;
            public GroundCorridor.Cursor Ground;
            public long Serial;
        }

        readonly V3 start, goal;
        readonly double within;
        readonly double arrivalLevel;
        readonly double initialEstimate;
        readonly Func<V3, double> guidance;
        readonly Func<V3, V3, V3?> probe;
        readonly Func<V3, bool> stagingEligible;
        readonly bool heightAware;
        Node staging;
        readonly int maxNodes;
        readonly double maxCost;
        readonly Dictionary<(int, int, int, int), double> costs = new Dictionary<(int, int, int, int), double>();
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
        public bool NodeLimitReached => Status == RouteStatus.NoRoute && Expanded >= maxNodes;
        /// <summary>A found path is only a checked leg, not arrival at the destination.</summary>
        public bool ProgressLeg { get; }
        // Only expanded nodes, reached by checked edges, can offer staging.
        // This is not a path to the real goal and never changes Status.
        public IReadOnlyList<V3> StagingPath => staging == null ? null : PathTo(staging, staging.Pos);

        public LocalRoute(V3 start, V3 goal, Func<V3, V3, V3?> probe, double within = .25,
            bool groundCorridor = false, double maxCost = double.PositiveInfinity, int maxNodes = MaxNodes,
            double progressHorizon = FrontierDistance * 1.5, double arrivalLevel = Arrival.Level,
            Func<V3, double> guidance = null, Func<V3, bool> stagingEligible = null, bool heightAware = false)
        {
            if (maxNodes < 1 || maxNodes > MaxNodes) throw new ArgumentOutOfRangeException(nameof(maxNodes));
            if (double.IsNaN(maxCost) || maxCost <= 0) throw new ArgumentOutOfRangeException(nameof(maxCost));
            if (double.IsNaN(progressHorizon) || progressHorizon < FrontierDistance) throw new ArgumentOutOfRangeException(nameof(progressHorizon));
            this.start = start;
            this.goal = goal;
            this.probe = probe;
            this.within = within;
            this.arrivalLevel = arrivalLevel;
            this.maxNodes = maxNodes; this.maxCost = maxCost;
            this.guidance = guidance;
            this.stagingEligible = stagingEligible;
            initialEstimate = Remaining(start);
            // Keep taking checked approach legs until the destination is
            // within the local horizon. Starting a full floor/entrance search
            // from 24 m away can spend its entire bound on the lower bank.
            ProgressLeg = initialEstimate > progressHorizon;
            this.heightAware = heightAware && !ProgressLeg && guidance == null;
            Add(new Node { Pos = start, Ground = groundCorridor ? new GroundCorridor.Cursor() : null });
        }

        public RouteStatus Advance(int budget = 8)
        {
            for (var i = 0; i < budget && Status == RouteStatus.Searching; i++)
            {
                if (open.Count == 0 || Expanded >= maxNodes) { Status = RouteStatus.NoRoute; break; }
                var node = open.Min;
                open.Remove(node);
                if (costs[Key(node)] < node.Cost) { i--; continue; }
                Expanded++;
                if (stagingEligible != null && node.Cost <= DryStaging.MaximumCost &&
                    V3.FlatDistance(start, node.Pos) >= DryStaging.MinimumDistance &&
                    V3.FlatDistance(start, goal) - V3.FlatDistance(node.Pos, goal) >= DryStaging.MinimumProgress &&
                    (staging == null || node.Remaining < staging.Remaining) && stagingEligible(node.Pos)) staging = node;
                if (Arrival.Reached(node.Pos, goal, within, arrivalLevel))
                {
                    Complete(node, node.Pos);
                    break;
                }
                // Plan only through loaded, probed ground. A distant goal's
                // height may belong to an unloaded hill or raised floor;
                // intermediate legs follow support, never that final height.
                if (ProgressLeg && V3.FlatDistance(start, node.Pos) >= FrontierDistance &&
                    initialEstimate - node.Remaining >= FrontierProgress)
                {
                    Complete(node, node.Pos);
                    break;
                }
                if (V3.FlatDistance(node.Pos, goal) <= within + Cell * 1.5 && Math.Abs(node.Pos.Y - goal.Y) <= arrivalLevel)
                {
                    Probes++;
                    // Grid cells can straddle the narrow space beside a
                    // solid goal. Probe a reachable arrival point, not the
                    // occupied centre of a chest/tree/other target.
                    var offset = (node.Pos - goal).Flat.Normalized * (within * .95);
                    var end = Step(node, new V3(goal.X + offset.X, node.Pos.Y, goal.Z + offset.Z), out _);
                    if (end is V3 at && node.Cost + (at - node.Pos).Length <= maxCost && Arrival.Reached(at, goal, within, arrivalLevel))
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
                        var point = Step(node, desired, out var ground);
                        if (!(point is V3 pos)) continue;
                        var cost = node.Cost + (pos - node.Pos).Length;
                        if (cost > maxCost) continue;
                        var next = new Node { X = nx, Z = nz, Pos = pos, Cost = cost, Parent = node, Ground = ground };
                        if (costs.TryGetValue(Key(next), out var previous) && previous <= cost + 1e-6) continue;
                        Add(next);
                    }
            }
            return Status;
        }

        V3? Step(Node node, V3 desired, out GroundCorridor.Cursor ground)
        {
            ground = node.Ground?.Fork();
            if (ground == null) return probe(node.Pos, desired);
            return ground.Inspect(node.Pos, desired, probe) == null ? ground.At : null;
        }

        // A safe arrival grade can continue where another approach cannot.
        // Retain both rather than pruning them as the same grid cell/floor.
        static (int, int, int, int) Key(Node n) => (n.X, n.Z, (int)Math.Round(n.Pos.Y * 4),
            n.Ground == null ? 0 : n.Ground.LastAngle.HasValue ? 1 + (int)Math.Round(n.Ground.LastAngle.Value * 10) : int.MinValue);

        void Complete(Node node, V3 at)
        {
            Path = PathTo(node, at);
            Status = RouteStatus.Found;
        }

        static IReadOnlyList<V3> PathTo(Node node, V3 at)
        {
            var path = new List<V3> { at };
            for (var p = node; p != null; p = p.Parent) path.Add(p.Pos);
            path.Reverse();
            return path;
        }

        void Add(Node node)
        {
            costs[Key(node)] = node.Cost;
            node.Remaining = Remaining(node.Pos);
            node.Score = node.Cost + node.Remaining;
            node.Serial = serial++;
            open.Add(node);
        }

        double Remaining(V3 at)
        {
            if (heightAware)
            {
                // Distance to the arrival cylinder is a lower bound on the
                // checked 3D path cost. Prefer the required floor without
                // inventing a connection to it. Distant frontier legs and
                // explicit regional guidance keep their existing estimate.
                var horizontal = Math.Max(0, V3.FlatDistance(at, goal) - within);
                var vertical = Math.Max(0, Math.Abs(at.Y - goal.Y) - arrivalLevel);
                return Math.Sqrt(horizontal * horizontal + vertical * vertical);
            }
            var estimate = guidance != null ? guidance(at) : V3.FlatDistance(at, goal);
            if (double.IsNaN(estimate) || estimate < 0) throw new ArgumentException("invalid route guidance");
            return estimate;
        }
    }
}
