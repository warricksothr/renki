using System.Collections.Generic;
using System.Diagnostics;
using Renki.Perception;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// What is near the bot (docs/BRIDGE-PROTOCOL.md 6.2): characters,
    /// the structures it might use, and things it could harvest, within
    /// Radius, nearest first, at most MaxPerKind of each kind. Runs from Update
    /// at perceptionHz, never per physics step, and times every scan.
    /// </summary>
    sealed class Perceiver
    {
        public double Radius = 40;
        public int MaxPerKind = 24;

        /// <summary>How often the scan time goes to the audit log as a trace.</summary>
        const float ReportEvery = 30f;

        // The layers trees, logs, rocks, pickables and drops sit on. Pieces
        // and characters have their own lists in the game.
        static readonly int ResourceMask = LayerMask.GetMask("Default", "static_solid", "Default_small", "piece_nonsolid", "item");

        readonly List<Character> characterBuffer = new List<Character>();
        readonly List<Piece> pieceBuffer = new List<Piece>();
        readonly Collider[] colliders = new Collider[2048];
        readonly HashSet<ZNetView> seen = new HashSet<ZNetView>();
        readonly NearestByKind<PerceivedCharacter> characters = new NearestByKind<PerceivedCharacter>(24);
        readonly NearestByKind<PerceivedStructure> structures = new NearestByKind<PerceivedStructure>(24);
        readonly NearestByKind<PerceivedResource> resources = new NearestByKind<PerceivedResource>(24);

        /// <summary>Past this many, only the nearest walls go in a layout.</summary>
        const int MaxWalls = 512;
        readonly Nearest<KeyValuePair<int, Wall>> walls = new Nearest<KeyValuePair<int, Wall>>(MaxWalls);
        ulong sentWalls;
        bool wallsSent;
        Layout layout;

        readonly Stopwatch watch = new Stopwatch();
        int scans;
        double totalMs, worstMs;
        int mostColliders;
        float nextReport;

        public Protocol.Perception Scan(Player me)
        {
            watch.Restart();
            characters.Cap = structures.Cap = resources.Cap = MaxPerKind;
            var at = me.transform.position;
            var r = (float)Radius;

            characters.Clear();
            characterBuffer.Clear();
            Character.GetCharactersInRange(at, r, characterBuffer);
            foreach (var c in characterBuffer)
            {
                if (c == me || c.IsDead()) continue;
                var pc = Describe(c, me);
                characters.Add(pc.Kind, pc, Vector3.Distance(at, c.transform.position));
            }

            structures.Clear();
            walls.Clear();
            measured = 0;
            pieceBuffer.Clear();
            Piece.GetAllPiecesInRadius(at, r, pieceBuffer);
            foreach (var p in pieceBuffer)
            {
                var s = Describe(p, out var wall);
                if (s != null) structures.Add(s.Kind, s, Vector3.Distance(at, p.transform.position));
                else if (wall != null) walls.Add(new KeyValuePair<int, Wall>(p.GetInstanceID(), wall), Vector3.Distance(at, p.transform.position));
            }
            Lay();

            resources.Clear();
            seen.Clear();
            var hits = Physics.OverlapSphereNonAlloc(at, r, colliders, ResourceMask);
            if (hits > mostColliders) mostColliders = hits;
            for (var i = 0; i < hits; i++)
            {
                var res = Describe(colliders[i], out var pos);
                if (res != null) resources.Add(res.Kind, res, Vector3.Distance(at, pos));
            }

            var snapshot = new Protocol.Perception
            {
                Characters = characters.Take(),
                Structures = structures.Take(),
                Resources = resources.Take(),
            };
            watch.Stop();
            Account(snapshot, hits);
            return snapshot;
        }

        static PerceivedCharacter Describe(Character c, Player me)
        {
            var pc = new PerceivedCharacter
            {
                Id = c.GetZDOID().ToString(),
                Pos = Convert.ToArray(c.transform.position),
                HpFrac = System.Math.Round(c.GetHealthPercentage(), 2),
            };
            if (c is Player p)
            {
                pc.Kind = Wire.PerceivedPlayer;
                pc.Name = p.GetPlayerName();
                return pc;
            }
            pc.Kind = c.IsTamed() ? Wire.PerceivedTamed : Wire.PerceivedCreature;
            pc.Prefab = PrefabName(c.m_nview, c.gameObject);
            if (c.GetLevel() > 1) pc.Stars = c.GetLevel() - 1;
            if (BaseAI.IsEnemy(c, me)) pc.Hostile = true;
            var ai = c.GetBaseAI();
            if (ai != null && ai.GetTargetCreature() == me) pc.TargetingMe = true;
            return pc;
        }

        /// <summary>What a piece or a harvestable object is never changes, so
        /// it is worked out once and kept; null means the bot has no use for
        /// it. Only what can change (a door opening, a bush picked) is read
        /// on every scan.</summary>
        sealed class Known
        {
            public string Kind, Prefab;
            public Component Part;
            public int? MinTool;
            /// <summary>For a wall, its foot line and height in the piece's
            /// own frame; Kind is then null.</summary>
            public (V3 A, V3 B, double H)? Wall;
        }

        /// <summary>Past this many, the caches start again, so destroyed
        /// objects do not pile up.</summary>
        const int CacheMax = 8192;

        /// <summary>A piece that is not a structure and has yet to be
        /// measured for a wall.</summary>
        static readonly Known Unmeasured = new Known();

        /// <summary>How many new pieces one scan measures for walls.</summary>
        const int MeasurePerScan = 200;
        int measured;

        readonly Dictionary<Piece, Known> pieces = new Dictionary<Piece, Known>();
        readonly Dictionary<Collider, ZNetView> owners = new Dictionary<Collider, ZNetView>();
        readonly Dictionary<ZNetView, Known> objects = new Dictionary<ZNetView, Known>();

        /// <summary>The piece as 6.2 names it, or null for the walls, floors
        /// and everything else the bot has no use for; a wall comes back in
        /// wall instead, in world space (6.9).</summary>
        PerceivedStructure Describe(Piece p, out Wall wall)
        {
            wall = null;
            if (!pieces.TryGetValue(p, out var k))
            {
                if (pieces.Count >= CacheMax) pieces.Clear();
                pieces[p] = k = Classify(p) ?? Unmeasured;
            }
            if (k == Unmeasured)
            {
                // Measuring a piece is the costly part: a big base is
                // measured over a few scans rather than in one frame.
                if (measured >= MeasurePerScan) return null;
                measured++;
                pieces[p] = k = WallOf(p) is (V3, V3, double) shape ? new Known { Wall = shape } : null;
            }
            if (k == null) return null;
            if (k.Wall is (V3 a, V3 b, double h))
            {
                var t = p.transform;
                wall = new Wall
                {
                    A = Convert.ToArray(t.TransformPoint(Convert.ToVector3(a))),
                    B = Convert.ToArray(t.TransformPoint(Convert.ToVector3(b))),
                    H = System.Math.Round(h, 2),
                };
                return null;
            }
            var s = new PerceivedStructure { Kind = k.Kind, Prefab = k.Prefab, Pos = Convert.ToArray(p.transform.position), Id = Id(p.m_nview) };
            switch (k.Part)
            {
                case Door door:
                    var zdo = door.m_nview != null ? door.m_nview.GetZDO() : null;
                    s.Open = zdo != null && zdo.GetInt(ZDOVars.s_state, 0) != 0;
                    break;
                case Bed bed:
                    if (bed.IsMine()) s.Mine = true;
                    break;
                case TeleportWorld portal:
                    var tag = portal.GetText();
                    if (!string.IsNullOrEmpty(tag)) s.Tag = tag;
                    break;
            }
            return s;
        }

        static Known Classify(Piece p)
        {
            Component part;
            string kind;
            if (p.TryGetComponent<Door>(out var door)) (kind, part) = (Wire.StructureDoor, door);
            else if (p.TryGetComponent<Bed>(out var bed)) (kind, part) = (Wire.StructureBed, bed);
            else if (p.TryGetComponent<TeleportWorld>(out var portal)) (kind, part) = (Wire.StructurePortal, portal);
            else if (p.TryGetComponent<CraftingStation>(out var station)) (kind, part) = (Wire.StructureStation, station);
            else if (p.TryGetComponent<Smelter>(out var smelter)) (kind, part) = (Wire.StructureSmelter, smelter);
            else if (p.TryGetComponent<Fireplace>(out var fire)) (kind, part) = (Wire.StructureFire, fire);
            else if (p.TryGetComponent<Container>(out var chest)) (kind, part) = (Wire.StructureContainer, chest);
            else return null;
            return new Known { Kind = kind, Part = part, Prefab = PrefabName(p.m_nview, p.gameObject) };
        }

        static readonly List<Collider> pieceColliders = new List<Collider>();

        /// <summary>Whether the piece is a wall, from the box its solid
        /// colliders fill in its own frame.</summary>
        static (V3, V3, double)? WallOf(Piece p)
        {
            var root = p.transform;
            Vector3 min = Vector3.positiveInfinity, max = Vector3.negativeInfinity;
            pieceColliders.Clear();
            p.GetComponentsInChildren(pieceColliders);
            foreach (var col in pieceColliders)
            {
                if (col.isTrigger) continue;
                Bounds box;
                if (col is BoxCollider bc) box = new Bounds(bc.center, bc.size);
                else if (col is MeshCollider mc && mc.sharedMesh != null) box = mc.sharedMesh.bounds;
                else continue;
                for (var i = 0; i < 8; i++)
                {
                    var corner = box.center + Vector3.Scale(box.extents,
                        new Vector3((i & 1) == 0 ? -1 : 1, (i & 2) == 0 ? -1 : 1, (i & 4) == 0 ? -1 : 1));
                    var local = root.InverseTransformPoint(col.transform.TransformPoint(corner));
                    min = Vector3.Min(min, local);
                    max = Vector3.Max(max, local);
                }
            }
            if (min.x > max.x) return null;
            return WallShape.Of(Convert.ToV3(min), Convert.ToV3(max));
        }

        /// <summary>The nearest walls, as a layout to send when they are not
        /// the ones last sent: walls stay put, so which pieces they are is
        /// enough to tell.</summary>
        void Lay()
        {
            var near = walls.Take();
            ulong sig = (ulong)near.Count;
            foreach (var w in near) unchecked { sig += Mix((ulong)(uint)w.Key); }
            if (wallsSent && sig == sentWalls) return;
            layout = new Layout();
            foreach (var w in near) layout.Walls.Add(w.Value);
            sentWalls = sig;
            wallsSent = true;
        }

        /// <summary>The splitmix64 finaliser: spreads an id over 64 bits, so
        /// sums of different sets of ids differ.</summary>
        static ulong Mix(ulong x)
        {
            unchecked
            {
                x += 0x9E3779B97F4A7C15UL;
                x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9UL;
                x = (x ^ (x >> 27)) * 0x94D049BB133111EBUL;
                return x ^ (x >> 31);
            }
        }

        /// <summary>The layout, once, after the scan that found it changed.</summary>
        public Layout TakeLayout()
        {
            var l = layout;
            layout = null;
            return l;
        }

        /// <summary>Send the layout again after the next scan, as after welcome.</summary>
        public void Resend() => wallsSent = false;

        /// <summary>What a collider belongs to, if the bot could harvest or
        /// pick it up; each object once, however many colliders it has.</summary>
        PerceivedResource Describe(Collider col, out Vector3 pos)
        {
            pos = default;
            if (!owners.TryGetValue(col, out var nview))
            {
                if (owners.Count >= CacheMax) owners.Clear();
                owners[col] = nview = col.GetComponentInParent<ZNetView>();
            }
            if (nview == null || !seen.Add(nview)) return null;
            if (!objects.TryGetValue(nview, out var k))
            {
                if (objects.Count >= CacheMax) objects.Clear();
                objects[nview] = k = Classify(nview);
            }
            if (k == null) return null;
            var r = new PerceivedResource { Kind = k.Kind, Prefab = k.Prefab, MinTool = k.MinTool };
            switch (k.Part)
            {
                case Pickable pick when pick.GetPicked():
                    return null;
                case ItemDrop drop when drop.m_itemData != null:
                    r.Stack = drop.m_itemData.m_stack;
                    break;
            }
            pos = nview.transform.position;
            r.Pos = Convert.ToArray(pos);
            r.Id = Id(nview);
            return r;
        }

        static Known Classify(ZNetView nview)
        {
            var go = nview.gameObject;
            var k = new Known { Prefab = PrefabName(nview, go) };
            if (go.TryGetComponent<TreeBase>(out var tree)) (k.Kind, k.MinTool) = (Wire.ResourceTree, tree.m_minToolTier);
            else if (go.TryGetComponent<TreeLog>(out var log)) (k.Kind, k.MinTool) = (Wire.ResourceLog, log.m_minToolTier);
            else if (go.TryGetComponent<MineRock>(out var rock)) (k.Kind, k.MinTool) = (Wire.ResourceRock, rock.m_minToolTier);
            else if (go.TryGetComponent<MineRock5>(out var rock5)) (k.Kind, k.MinTool) = (Wire.ResourceRock, rock5.m_minToolTier);
            // The game counts both stumps and small trees (Beech_small1) as
            // destructible trees; only the stumps are named for it.
            else if (go.TryGetComponent<Destructible>(out var wood) && wood.m_destructibleType == DestructibleType.Tree)
                (k.Kind, k.MinTool) = (k.Prefab.IndexOf("stub", System.StringComparison.OrdinalIgnoreCase) >= 0
                    ? Wire.ResourceStump : Wire.ResourceTree, wood.m_minToolTier);
            else if (go.TryGetComponent<Pickable>(out var pick)) (k.Kind, k.Part) = (Wire.ResourcePickable, pick);
            else if (go.TryGetComponent<ItemDrop>(out var drop)) (k.Kind, k.Part) = (Wire.ResourceDrop, drop);
            else return null;
            return k;
        }

        static string Id(ZNetView nview)
        {
            var zdo = nview != null ? nview.GetZDO() : null;
            return zdo != null ? zdo.m_uid.ToString() : "";
        }

        /// <summary>The prefab, from the network object's prefab hash where it
        /// has one: some objects rename their GameObject (MineRock5 to
        /// "___MineRock5 m_meshFilter").</summary>
        static string PrefabName(ZNetView nview, GameObject go)
        {
            var zdo = nview != null ? nview.GetZDO() : null;
            var prefab = zdo != null && ZNetScene.instance != null ? ZNetScene.instance.GetPrefab(zdo.GetPrefab()) : null;
            if (prefab != null) return prefab.name;
            var name = go.name;
            var clone = name.IndexOf("(Clone)", System.StringComparison.Ordinal);
            return (clone >= 0 ? name.Substring(0, clone) : name).Trim();
        }

        /// <summary>Keeps the scan time, and every ReportEvery seconds sends
        /// it with the counts as a trace, for the 2 ms budget.</summary>
        void Account(Protocol.Perception snapshot, int hits)
        {
            var ms = watch.Elapsed.TotalMilliseconds;
            scans++;
            totalMs += ms;
            if (ms > worstMs) worstMs = ms;
            if (Time.unscaledTime < nextReport) return;
            nextReport = Time.unscaledTime + ReportEvery;
            Tracer.Emit("perception", "scan", new Dictionary<string, object>
            {
                ["scans"] = scans,
                ["avgMs"] = System.Math.Round(totalMs / scans, 3),
                ["worstMs"] = System.Math.Round(worstMs, 3),
                ["characters"] = $"{snapshot.Characters.Count}/{characters.Seen}",
                ["structures"] = $"{snapshot.Structures.Count}/{structures.Seen}",
                ["resources"] = $"{snapshot.Resources.Count}/{resources.Seen}",
                ["colliders"] = hits,
                // The buffer filled: some objects in range were not looked at.
                ["truncated"] = mostColliders >= colliders.Length,
                ["mostColliders"] = mostColliders,
                ["pieces"] = pieceBuffer.Count,
                ["walls"] = walls.Seen,
            });
            scans = 0;
            totalMs = worstMs = 0;
            mostColliders = 0;
        }
    }
}
