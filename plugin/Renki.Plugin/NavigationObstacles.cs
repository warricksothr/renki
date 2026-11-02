using System;
using System.Collections.Generic;
using System.Diagnostics;
using Renki.Movement;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Bounded mark/refresh/sweep of static solids and transient log
    /// travel exclusions. Other dynamic/complex objects use live physics.</summary>
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
        public readonly LogFootprintIndex Logs;
        readonly V3[] corners = new V3[8];
        const int MaxLogs = 256;
        int logCount;
        float nextNear, nextAhead, nextFar, nextTrace;
        int pending, cursor;
        long serial;
        int rebuilt, removed, queries, truncated;
        double totalMs, worstMs;

        sealed class Entry
        {
            public int Id;
            public Collider Collider;
            public TreeLog Log;
            public bool IsLog;
            public LogFootprint? Footprint;
            public (Matrix4x4, Vector3, Vector3, float, float, int) Stamp;
            public float Seen;
            public (double, long, int)? Queued;
            public LinkedListNode<Entry> Sweep;
            public int CheckedFrame = -1;
            public bool Current;
        }

        public NavigationObstacles(int mask) { this.mask = mask; current = Current; Logs = new LogFootprintIndex(current); }
        public void Clear()
        {
            entries.Clear(); dirty.Clear(); sweep.Clear(); Index.Clear(); Logs.Clear(); logCount = 0;
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
                if (!Usable(entry.Collider) || entry.IsLog && entry.Log == null) { Remove(entry); continue; }
                // Discovery and rebuild may be on different frames. Stamp
                // the geometry actually measured, never the queued version.
                entry.Stamp = Stamp(entry.Collider); entry.CheckedFrame = -1;
                if (entry.IsLog)
                {
                    entry.Footprint = MeasureLog(entry.Collider);
                    if (entry.Footprint is LogFootprint footprint) Logs.Put(entry.Id, footprint);
                    else Logs.Remove(entry.Id);
                }
                else if (Shape(entry.Collider) is ObstacleShape shape) Index.Put(entry.Id, shape);
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
                    ["logs"] = Logs.Count,
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
            !c.isTrigger && (mask & (1 << c.gameObject.layer)) != 0 &&
            (IsLog(c) || c.attachedRigidbody == null && (c is BoxCollider || c is CapsuleCollider));

        static TreeLog Owner(Collider c) => c != null ? c.GetComponentInParent<TreeLog>() : null;
        static bool IsLog(Collider c) => Owner(c) != null;

        static (Matrix4x4, Vector3, Vector3, float, float, int) Stamp(Collider c)
        {
            var capsule = c as CapsuleCollider;
            var bounds = LocalBounds(c, out _);
            return (c.transform.localToWorldMatrix, bounds.center, bounds.size, capsule != null ? capsule.radius : 0,
                capsule != null ? capsule.height : 0, capsule != null ? capsule.direction :
                c is MeshCollider mesh && mesh.sharedMesh != null ? mesh.sharedMesh.GetInstanceID() : 0);
        }

        void Mark(Collider c, Vector3 bot, Vector3 toward, float now)
        {
            if (!Usable(c)) return;
            var id = c.GetInstanceID();
            var stamp = Stamp(c);
            if (entries.TryGetValue(id, out var e) && e.Collider != c) { Remove(e); e = null; }
            if (e == null)
            {
                var log = Owner(c);
                if (entries.Count >= MaxEntries || log != null && logCount >= MaxLogs) return;
                e = new Entry { Id = id, Collider = c, Log = log, IsLog = log != null };
                if (e.IsLog) logCount++;
                entries[id] = e; e.Sweep = sweep.AddLast(e);
            }
            e.Seen = now;
            if (!e.Stamp.Equals(stamp))
            {
                // Drop stale bounds immediately, then rebuild in priority
                // order. Unmeasured space is still checked by live physics.
                Index.Remove(id); Logs.Remove(id); e.Footprint = null; e.Stamp = stamp; e.CheckedFrame = -1;
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
                    (!e.IsLog || e.Log != null) &&
                    e.Collider.GetComponentInParent<Door>() == null;
            }
            return e.Current;
        }

        void Remove(Entry e)
        {
            Index.Remove(e.Id); Logs.Remove(e.Id); entries.Remove(e.Id); sweep.Remove(e.Sweep);
            if (e.IsLog) logCount--;
            if (e.Queued.HasValue) dirty.Remove(e.Queued.Value);
            removed++;
        }

        // Primitive/mesh local bounds preserve the log's orientation. Unknown
        // collider types fall back to their world envelope, explicitly kept
        // separate from solid primitives and always backed by live queries.
        static Bounds LocalBounds(Collider c, out bool worldBounds)
        {
            worldBounds = false;
            if (c is BoxCollider box) return new Bounds(box.center, box.size);
            if (c is MeshCollider mesh && mesh.sharedMesh != null) return mesh.sharedMesh.bounds;
            if (c is SphereCollider sphere) return new Bounds(sphere.center, Vector3.one * sphere.radius * 2);
            if (c is CapsuleCollider capsule)
            {
                var size = Vector3.one * capsule.radius * 2;
                size[capsule.direction] = Math.Max(capsule.height, capsule.radius * 2);
                return new Bounds(capsule.center, size);
            }
            worldBounds = true; return c.bounds;
        }

        LogFootprint? MeasureLog(Collider c)
        {
            var b = LocalBounds(c, out var worldBounds);
            var matrix = worldBounds ? Matrix4x4.identity : c.transform.localToWorldMatrix;
            // Unity scales a sphere by its largest axis and a capsule's
            // radius by the larger perpendicular axis. Transforming their
            // raw local bounds would understate nonuniformly scaled shapes.
            if (c is SphereCollider sphere)
            {
                var s = c.transform.lossyScale;
                var r = sphere.radius * Mathf.Max(Mathf.Abs(s.x), Mathf.Abs(s.y), Mathf.Abs(s.z));
                b = new Bounds(c.transform.TransformPoint(sphere.center), Vector3.one * r * 2);
                matrix = Matrix4x4.identity;
            }
            if (c is CapsuleCollider capsule)
            {
                var s = c.transform.lossyScale; var axis = capsule.direction;
                var r = capsule.radius * Mathf.Max(Mathf.Abs(s[(axis + 1) % 3]), Mathf.Abs(s[(axis + 2) % 3]));
                var size = Vector3.one * r * 2;
                size[axis] = Mathf.Max(capsule.height * Mathf.Abs(s[axis]), r * 2);
                b = new Bounds(Vector3.zero, size);
                matrix = Matrix4x4.TRS(c.transform.TransformPoint(capsule.center), c.transform.rotation, Vector3.one);
            }
            var heading = Vector3.right; var longest = 0f;
            for (var axis = 0; axis < 3; axis++)
            {
                var local = Vector3.zero; local[axis] = b.size[axis];
                var d = matrix.MultiplyVector(local); d.y = 0;
                if (d.sqrMagnitude > longest) { longest = d.sqrMagnitude; heading = d; }
            }
            for (var i = 0; i < 8; i++)
            {
                var p = b.center + Vector3.Scale(b.extents, new Vector3((i & 1) == 0 ? -1 : 1, (i & 2) == 0 ? -1 : 1, (i & 4) == 0 ? -1 : 1));
                corners[i] = Convert.ToV3(matrix.MultiplyPoint3x4(p));
            }
            return LogFootprint.Measure(corners, Convert.ToV3(heading));
        }

        public bool LeavingLog(Collider c, V3 from, V3 to)
        {
            if (!IsLog(c)) return false;
            var footprint = entries.TryGetValue(c.GetInstanceID(), out var e) && e.IsLog && e.Footprint.HasValue && Current(e.Id)
                ? e.Footprint : MeasureLog(c);
            return footprint.HasValue && footprint.Value.Leaving(from, to);
        }

        public List<LogObstacle> Snapshot()
        {
            var result = new List<LogObstacle>();
            foreach (var e in entries.Values)
            {
                if (!e.IsLog || !e.Footprint.HasValue || !Current(e.Id)) continue;
                var view = e.Log.GetComponent<ZNetView>();
                if (view == null || !view.IsValid()) continue;
                var f = e.Footprint.Value;
                result.Add(new LogObstacle
                {
                    Entity = view.GetZDO().m_uid.ToString(),
                    Collider = e.Id,
                    Outline = new[] { f.Corner(0).ToArray(), f.Corner(1).ToArray(), f.Corner(2).ToArray(), f.Corner(3).ToArray() },
                    MinY = f.MinY,
                    MaxY = f.MaxY,
                    Padding = LogFootprint.Padding
                });
            }
            return result;
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
