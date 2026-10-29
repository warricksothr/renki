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
            pieceBuffer.Clear();
            Piece.GetAllPiecesInRadius(at, r, pieceBuffer);
            foreach (var p in pieceBuffer)
            {
                var s = Describe(p);
                if (s != null) structures.Add(s.Kind, s, Vector3.Distance(at, p.transform.position));
            }

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
        }

        /// <summary>Past this many, the caches start again, so destroyed
        /// objects do not pile up.</summary>
        const int CacheMax = 8192;

        readonly Dictionary<Piece, Known> pieces = new Dictionary<Piece, Known>();
        readonly Dictionary<Collider, ZNetView> owners = new Dictionary<Collider, ZNetView>();
        readonly Dictionary<ZNetView, Known> objects = new Dictionary<ZNetView, Known>();

        /// <summary>The piece as 6.2 names it, or null for the walls, floors
        /// and everything else the bot has no use for.</summary>
        PerceivedStructure Describe(Piece p)
        {
            if (!pieces.TryGetValue(p, out var k))
            {
                if (pieces.Count >= CacheMax) pieces.Clear();
                pieces[p] = k = Classify(p);
            }
            if (k == null) return null;
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
            else if (go.TryGetComponent<Destructible>(out var stump) && stump.m_destructibleType == DestructibleType.Tree)
                (k.Kind, k.MinTool) = (Wire.ResourceStump, stump.m_minToolTier);
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
                ["mostColliders"] = mostColliders,
                ["pieces"] = pieceBuffer.Count,
            });
            scans = 0;
            totalMs = worstMs = 0;
            mostColliders = 0;
        }
    }
}
