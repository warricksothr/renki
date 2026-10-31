using System;

namespace Renki.Movement
{
    /// <summary>Execution evidence survives replanning. Only new physical
    /// forward progress buys more walking time; bobbing against a stair does not.</summary>
    public sealed class RouteProgress
    {
        public const double Window = 4;
        V3 anchor, direction, target;
        double best, effort, sampled = double.NaN;
        bool tracking;

        public void Reset() { tracking = false; sampled = double.NaN; best = effort = 0; }

        // attempted is the leg commanded on the preceding physics tick.
        // A null leg pauses the budget during planning, doors or deliberate holds.
        public bool Stalled(V3 pos, V3? attempted, double now)
        {
            var elapsed = double.IsNaN(sampled) ? 0 : Math.Max(0, Math.Min(.25, now - sampled));
            sampled = now;
            if (!attempted.HasValue) return false;
            var next = attempted.Value;
            var heading = (next - pos).Flat.Normalized;
            if (!tracking || (next - target).Length > 1.5 ||
                V3.Dot(direction, heading) < .25 || V3.FlatDistance(pos, anchor) > 3)
            {
                tracking = true; anchor = pos; direction = heading; target = next;
                best = effort = 0;
                return false;
            }
            var forward = V3.Dot((pos - anchor).Flat, direction);
            if (forward >= best + .2) { best = forward; effort = 0; }
            else effort += elapsed;
            if (effort < Window) return false;
            // The caller remembers this refusal and replans; do not charge
            // another failed leg until it has actually commanded movement.
            tracking = false; effort = 0;
            return true;
        }
    }
}
