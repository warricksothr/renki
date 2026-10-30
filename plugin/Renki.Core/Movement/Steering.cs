using System;
using System.Collections.Generic;
using Renki.Protocol;

namespace Renki.Movement
{
    /// <summary>What the stuck detector wants done next.</summary>
    public enum Recovery { None, Jump, Strafe, Repath, Climb, GiveUp }

    /// <summary>
    /// Stuck detection (docs/BRIDGE-PROTOCOL.md 7.3): the bot is stuck while
    /// it stays within Radius metres of where it was. Each Window seconds
    /// inside that circle triggers the next recovery: a jump, then a strafe,
    /// then a re-path. That is one round; after Rounds rounds the task gives
    /// up. After the first round a climb (Climber) takes the jump's place:
    /// one hop does not clear what the first jump did not. Hopping or shuffling against a wall does not start the count
    /// again. Once a recovery has fired, leaving the circle only starts a new
    /// one; the count goes back to nothing when the bot gets Escape metres
    /// from where the first recovery fired, so a strafe that carries it just
    /// over the edge and back is still stuck.
    /// </summary>
    public sealed class StuckDetector
    {
        public double Window = 3;
        public double Radius = 3;
        public double Escape = 6;
        public int Rounds = 3;

        static readonly Recovery[] Sequence = { Recovery.Jump, Recovery.Strafe, Recovery.Repath };
        static readonly Recovery[] Later = { Recovery.Climb, Recovery.Strafe, Recovery.Repath };

        V3 center;
        V3 anchor;
        double since = double.NaN;
        int step;

        public void Reset(V3 pos, double now)
        {
            center = pos;
            since = now;
            step = 0;
        }

        /// <summary>Call every tick while the bot is trying to move.</summary>
        public Recovery Update(V3 pos, double now)
        {
            if (double.IsNaN(since))
            {
                Reset(pos, now);
                return Recovery.None;
            }
            if (V3.FlatDistance(pos, center) > Radius)
            {
                if (step == 0 || V3.FlatDistance(pos, anchor) > Escape) step = 0;
                center = pos;
                since = now;
                return Recovery.None;
            }
            if (now - since < Window) return Recovery.None;
            if (step >= Sequence.Length * Rounds) return Recovery.GiveUp;
            if (step == 0) anchor = center;
            since = now;
            var round = step / Sequence.Length == 0 ? Sequence : Later;
            return round[step++ % Sequence.Length];
        }

        /// <summary>Holds the count where it is while a recovery takes its
        /// time, as a climb waiting for stamina does: the bot is not moving
        /// on purpose, and that is not another round of being stuck.</summary>
        public void Hold(double now) => since = now;

        /// <summary>The middle of the circle the bot has to leave.</summary>
        public V3 Center => center;

        /// <summary>How many recoveries have been tried since the last progress.</summary>
        public int Attempts => step;
    }

    /// <summary>Walks a list of waypoints.</summary>
    public sealed class PathFollower
    {
        readonly List<V3> points = new List<V3>();
        int index;

        /// <summary>How close counts as reaching a waypoint.</summary>
        public double Reach = 1.0;

        /// <summary>Optional tighter final tolerance for a precise interaction.
        /// Does not extend a partial path or loosen its existing tolerance.</summary>
        public double FinalReach = double.PositiveInfinity;

        public void Set(IEnumerable<V3> path)
        {
            points.Clear();
            points.AddRange(path);
            index = 0;
        }

        public int Count => points.Count;

        /// <summary>Never invent a segment beyond a partial path's end.</summary>
        public V3 Direction(V3 pos) => Next(pos) is V3 next ? (next - pos).Flat : V3.Zero;

        /// <summary>The waypoint to head for from pos, skipping any already
        /// reached on their own level, or null once the last is reached.
        /// Standing below a floor never consumes its waypoints.</summary>
        public V3? Next(V3 pos)
        {
            while (index < points.Count)
            {
                var last = index == points.Count - 1;
                var within = last ? Math.Min(Reach * 0.5, FinalReach) : Reach;
                if (!Arrival.Reached(pos, points[index], within)) return points[index];
                if (!last)
                {
                    index++;
                    continue;
                }
                return null;
            }
            return null;
        }
    }

    public enum Gait { Stop, Walk, Run }

    /// <summary>
    /// follow's pace (docs/BRIDGE-PROTOCOL.md 7.3): stop inside the distance,
    /// walk while the gap is under twice the distance, run beyond that, and
    /// in auto mode run whenever the leader runs. Once stopped, the bot waits
    /// for the gap to open by Slack before moving again, so it does not
    /// shuffle at the edge.
    /// </summary>
    public sealed class FollowPace
    {
        public double Slack = 1.0;
        bool stopped = true;

        public Gait Decide(double gap, double distance, RunMode mode, bool leaderRunning, bool blocked = false)
        {
            if (gap <= distance || ((stopped || blocked) && gap <= distance + Slack))
            {
                stopped = true;
                return Gait.Stop;
            }
            stopped = false;
            switch (mode)
            {
                case RunMode.Walk: return Gait.Walk;
                case RunMode.Run: return Gait.Run;
            }
            return gap > 2 * distance || leaderRunning ? Gait.Run : Gait.Walk;
        }
    }

    public static class Controls
    {
        /// <summary>
        /// Player.SetControls takes its move direction relative to where the
        /// character looks: z forward, x right. This turns a world direction
        /// into those axes for a given look direction, so the bot can walk one
        /// way while looking another.
        /// </summary>
        public static V3 ToLocal(V3 worldDir, V3 lookDir)
        {
            var fwd = lookDir.Flat.Normalized;
            if (fwd.Length < 0.5) fwd = V3.Forward;
            var right = new V3(fwd.Z, 0, -fwd.X); // Vector3.Cross(Vector3.up, fwd)
            var w = worldDir.Flat;
            var local = new V3(V3.Dot(w, right), 0, V3.Dot(w, fwd));
            return local.Length > 1 ? local.Normalized : local;
        }

        /// <summary>Yaw in degrees for a direction: 0 is +Z (north), clockwise.</summary>
        public static double Yaw(V3 dir)
        {
            var deg = Math.Atan2(dir.X, dir.Z) * 180 / Math.PI;
            return deg < 0 ? deg + 360 : deg;
        }
    }
}
