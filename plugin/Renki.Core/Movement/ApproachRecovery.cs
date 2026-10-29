using System;
using System.Collections.Generic;

namespace Renki.Movement
{
    public enum ApproachStep { None, Detour, Retry, GiveUp }

    /// <summary>A partial path beneath a goal is worth trying from another
    /// position. Walk away, then around it on either side, re-planning the
    /// real goal after each leg. The host must validate each detour's path.</summary>
    public sealed class ApproachRecovery
    {
        public const int Limit = 3;
        public const double LegSeconds = 6;
        public const double TotalSeconds = 30;
        public V3? Target { get; private set; }
        public int Attempts { get; private set; }

        double started = double.NaN, legStarted, retryAt;
        V3 away;

        public static bool SafePath(V3 from, V3 candidate, IReadOnlyList<V3> path, Func<V3, V3, bool> blocked)
        {
            if (path.Count == 0 || !Arrival.Reached(path[path.Count - 1], candidate, 1)) return false;
            foreach (var point in path)
            {
                if (blocked(from, point)) return false;
                from = point;
            }
            return true;
        }

        public void Reset()
        {
            Target = null;
            Attempts = 0;
            started = double.NaN;
            retryAt = 0;
        }

        /// <param name="project">Return a reachable, supported point near
        /// the candidate, or null for an incomplete or unsafe path.</param>
        public ApproachStep Update(V3 pos, V3 goal, bool atEnd, double now, Func<V3, V3?> project)
        {
            if (!double.IsNaN(started) && now - started >= TotalSeconds) return ApproachStep.GiveUp;
            if (Target is V3 target)
            {
                if ((pos - target).Length > 1 && now - legStarted < LegSeconds) return ApproachStep.None;
                Target = null;
                retryAt = now + 2;
                return ApproachStep.Retry;
            }
            if (!atEnd || now < retryAt) return ApproachStep.None;
            if (Attempts >= Limit) return ApproachStep.GiveUp;
            if (double.IsNaN(started))
            {
                started = now;
                away = (pos - goal).Flat.Normalized;
                if (away.Length < 0.5) away = V3.Forward;
            }

            var angle = Attempts == 0 ? 0 : Attempts == 1 ? Math.PI / 2 : -Math.PI / 2;
            Attempts++;
            var direction = new V3(away.X * Math.Cos(angle) - away.Z * Math.Sin(angle), 0,
                away.X * Math.Sin(angle) + away.Z * Math.Cos(angle));
            // A second radius helps when the first ring falls inside a wall.
            foreach (var radius in new[] { 6.0, 10.0 })
            {
                var candidate = new V3(goal.X, pos.Y, goal.Z) + direction * radius;
                var reachable = project(candidate);
                if (reachable is V3 point && V3.FlatDistance(pos, point) >= 2)
                {
                    Target = point;
                    legStarted = now;
                    return ApproachStep.Detour;
                }
            }
            // Tiles may still be building. Retry with a pause even when no
            // safe detour exists; never substitute a blind straight line.
            retryAt = now + 2;
            return ApproachStep.Retry;
        }
    }
}
