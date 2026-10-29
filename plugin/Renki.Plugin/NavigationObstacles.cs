using System;
using System.Collections.Generic;
using System.Diagnostics;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Bounded mark/refresh/sweep of static solid primitives. Meshes,
    /// doors and rigidbodies keep using the live physics path.</summary>
    sealed class NavigationObstacles
    {
        const int MaxEntries = 4096;
        readonly int mask;
        readonly Collider[] found = new Collider[2048];
        readonly Dictionary<int, Entry> entries = new Dictionary<int, Entry>();
        readonly SortedSet<(double Priority, long Serial, int Id)> dirty = new SortedSet<(double, long, int)>();
        readonly LinkedList<Entry> sweep = new LinkedList<Entry>();
        readonly Stopwatch watch = new Stopwatch();
        readonly Func<int, bool> current;
        public readonly ObstacleIndex Index = new ObstacleIndex();
        float nextNear, nextAhead, nextFar, nextTrace;
        int pending, cursor;
        long serial;
        int rebuilt, removed, queries, truncated;
        double totalMs, worstMs;

        sealed class Entry
        {
            public int Id;
            public Collider Collider;
            public (Matrix4x4, Vector3, Vector3, float, float, int) Stamp;
            public float Seen;
            public (double, long, int)? Queued;
            public LinkedListNode<Entry> Sweep;
            public int CheckedFrame = -1;
            public bool Current;
        }

        public NavigationObstacles(int mask) { this.mask = mask; current = Current; }
        public void Clear()
        {
            entries.Clear(); dirty.Clear(); sweep.Clear(); Index.Clear();
            nextNear = nextAhead = nextFar = 0;
            // NonAlloc buffers otherwise retain references to unloaded scenes.
            Array.Clear(found, 0, found.Length); pending = cursor = 0;
        }

        public void Update(Vector3 bot, Vector3 toward, float now)
        {
            watch.Restart();
            // At most one broad-phase query per frame. Process its results
            // incrementally before reusing the buffer, even if it saturated.
            if (cursor >= pending)
            {
                // Oldest due pass wins, so a saturated near scan cannot
                // permanently starve forward/far discovery.
                if (now >= nextNear && nextNear <= nextAhead && nextNear <= nextFar)
                { Discover(bot, 10); nextNear = now + .25f; }
                else if (now >= nextAhead && nextAhead <= nextFar)
                {
                    var dir = toward - bot; dir.y = 0;
                    Discover(bot + dir.normalized * Math.Min(16, dir.magnitude), 12);
                    nextAhead = now + .5f;
                }
                else if (now >= nextFar) { Discover(bot, 40); nextFar = now + 2; }
            }
            // Discovery uses Unity's existing broad phase; measuring changed
            // geometry is queued, not repeated for every discovered object.
            var maintenance = Stopwatch.StartNew();
            for (var n = 0; n < 128 && cursor < pending && maintenance.Elapsed.TotalMilliseconds < .5; n++)
            {
                var c = found[cursor]; found[cursor++] = null;
                Mark(c, bot, toward, now);
            }
            for (var n = 0; n < 16 && dirty.Count > 0 && maintenance.Elapsed.TotalMilliseconds < .75; n++)
            {
                var key = dirty.Min; dirty.Remove(key);
                if (!entries.TryGetValue(key.Id, out var entry)) continue;
                entry.Queued = null;
                if (!Usable(entry.Collider)) { Remove(entry); continue; }
                // Discovery and rebuild may be on different frames. Stamp
                // the geometry actually measured, never the queued version.
                entry.Stamp = Stamp(entry.Collider); entry.CheckedFrame = -1;
                if (Shape(entry.Collider) is ObstacleShape shape) Index.Put(entry.Id, shape);
                else Index.Remove(entry.Id);
                rebuilt++;
            }
            // Round-robin sweep is bounded even in a large base. Eviction
            // merely falls back to physics; it never declares a cell free.
            for (var n = 0; n < 32 && sweep.First != null; n++)
            {
                var node = sweep.First; sweep.RemoveFirst(); sweep.AddLast(node);
                var e = node.Value;
                if (e.Collider == null || !e.Collider.enabled || !e.Collider.gameObject.activeInHierarchy ||
                    now - e.Seen > 10 || (e.Collider.bounds.center - bot).sqrMagnitude > 64 * 64) Remove(e);
            }
            watch.Stop(); totalMs += watch.Elapsed.TotalMilliseconds; worstMs = Math.Max(worstMs, watch.Elapsed.TotalMilliseconds);
            if (now >= nextTrace)
            {
                Tracer.Emit("navigation", "obstacle cache", new Dictionary<string, object>
                {
                    ["tracked"] = entries.Count,
                    ["indexed"] = Index.Count,
                    ["dirty"] = dirty.Count,
                    ["rebuilt"] = rebuilt,
                    ["removed"] = removed,
                    ["queries"] = queries,
                    ["truncated"] = truncated,
                    ["ms"] = Math.Round(totalMs, 2),
                    ["worstFrameMs"] = Math.Round(worstMs, 2)
                });
                rebuilt = removed = queries = truncated = 0; totalMs = worstMs = 0; nextTrace = now + 30;
            }
        }

        void Discover(Vector3 center, float radius)
        {
            queries++;
            pending = Physics.OverlapSphereNonAlloc(center, radius, found, mask, QueryTriggerInteraction.Ignore);
            cursor = 0;
            if (pending == found.Length) truncated++;
        }

        bool Usable(Collider c) => c != null && c.enabled && c.gameObject.activeInHierarchy &&
            !c.isTrigger && c.attachedRigidbody == null && (mask & (1 << c.gameObject.layer)) != 0 &&
            (c is BoxCollider || c is CapsuleCollider);

        static (Matrix4x4, Vector3, Vector3, float, float, int) Stamp(Collider c)
        {
            var box = c as BoxCollider; var capsule = c as CapsuleCollider;
            return (c.transform.localToWorldMatrix, box != null ? box.center : capsule.center,
                box != null ? box.size : Vector3.zero, capsule != null ? capsule.radius : 0,
                capsule != null ? capsule.height : 0, capsule != null ? capsule.direction : 0);
        }

        void Mark(Collider c, Vector3 bot, Vector3 toward, float now)
        {
            if (!Usable(c)) return;
            var id = c.GetInstanceID();
            var stamp = Stamp(c);
            if (entries.TryGetValue(id, out var e) && e.Collider != c) { Remove(e); e = null; }
            if (e == null)
            {
                if (entries.Count >= MaxEntries) return;
                e = new Entry { Id = id, Collider = c }; entries[id] = e; e.Sweep = sweep.AddLast(e);
            }
            e.Seen = now;
            if (!e.Stamp.Equals(stamp))
            {
                // Drop stale bounds immediately, then rebuild in priority
                // order. Unmeasured space is still checked by live physics.
                Index.Remove(id); e.Stamp = stamp; e.CheckedFrame = -1;
            }
            else if (!e.Queued.HasValue) return;
            // Pending work follows the current bot and heading, even when
            // the object's geometry itself has not changed.
            Queue(e, bot, toward);
        }

        void Queue(Entry e, Vector3 bot, Vector3 toward)
        {
            if (e.Queued.HasValue) dirty.Remove(e.Queued.Value);
            var key = (ObstacleIndex.Priority(Convert.ToV3(bot), Convert.ToV3(toward), Convert.ToV3(e.Collider.bounds.center)), ++serial, e.Id);
            e.Queued = key; dirty.Add(key);
        }

        public bool Blocked(V3 from, V3 to) => Index.Blocked(from, to, current);

        bool Current(int id)
        {
            if (!entries.TryGetValue(id, out var e)) return false;
            if (e.CheckedFrame != Time.frameCount)
            {
                e.CheckedFrame = Time.frameCount;
                e.Current = Usable(e.Collider) && e.Stamp.Equals(Stamp(e.Collider)) &&
                    e.Collider.GetComponentInParent<Door>() == null;
            }
            return e.Current;
        }

        void Remove(Entry e)
        {
            Index.Remove(e.Id); entries.Remove(e.Id); sweep.Remove(e.Sweep);
            if (e.Queued.HasValue) dirty.Remove(e.Queued.Value);
            removed++;
        }

        static ObstacleShape? Shape(Collider c)
        {
            if (c.isTrigger || c.attachedRigidbody != null || c.GetComponentInParent<Door>() != null) return null;
            var t = c.transform;
            // Tilted boxes include stairs; their AABB is not a solid prism.
            if (Vector3.Dot(t.up, Vector3.up) < .9999f) return null;
            var right = t.TransformVector(Vector3.right);
            var up = t.TransformVector(Vector3.up);
            var forward = t.TransformVector(Vector3.forward);
            // A rotated child of a nonuniformly scaled parent can be
            // sheared. lossyScale/eulerAngles would invent solid corners.
            if (Math.Abs(Vector3.Dot(right.normalized, forward.normalized)) > .00001f ||
                Math.Abs(Vector3.Dot(right.normalized, up.normalized)) > .00001f ||
                Math.Abs(Vector3.Dot(forward.normalized, up.normalized)) > .00001f ||
                Vector3.Dot(up.normalized, Vector3.up) < .99999f ||
                Math.Abs(right.normalized.y) > .000001f || Math.Abs(forward.normalized.y) > .000001f) return null;
            var scale = new Vector3(right.magnitude, up.magnitude, forward.magnitude); var bounds = c.bounds;
            if (scale.x < .0001f || scale.y < .0001f || scale.z < .0001f) return null;
            if (bounds.extents.x > 16 || bounds.extents.z > 16) return null;
            if (c is BoxCollider box)
                return new ObstacleShape(Convert.ToV3(t.TransformPoint(box.center)),
                    Math.Abs(box.size.x * scale.x) * .5, Math.Abs(box.size.z * scale.z) * .5,
                    bounds.min.y, bounds.max.y, Math.Atan2(right.z, right.x));
            if (c is CapsuleCollider cap && cap.direction == 1 && Math.Abs(Math.Abs(scale.x) - Math.Abs(scale.z)) < .001f)
            {
                var radius = cap.radius * Math.Max(Math.Abs(scale.x), Math.Abs(scale.z));
                var center = t.TransformPoint(cap.center);
                var half = Math.Max(0, cap.height * Math.Abs(scale.y) * .5 - radius);
                if (half > .01)
                    return new ObstacleShape(Convert.ToV3(center), radius, radius, center.y - half, center.y + half, circle: true);
            }
            return null;
        }
    }
}
