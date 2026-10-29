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
    /// Radius, nearest first, at most MaxPerKind of each. Runs from Update
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
        readonly Nearest<PerceivedCharacter> characters = new Nearest<PerceivedCharacter>(24);
        readonly Nearest<PerceivedStructure> structures = new Nearest<PerceivedStructure>(24);
        readonly Nearest<PerceivedResource> resources = new Nearest<PerceivedResource>(24);

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
                characters.Add(Describe(c, me), Vector3.Distance(at, c.transform.position));
            }

            structures.Clear();
            pieceBuffer.Clear();
            Piece.GetAllPiecesInRadius(at, r, pieceBuffer);
            foreach (var p in pieceBuffer)
            {
                var s = Describe(p);
                if (s != null) structures.Add(s, Vector3.Distance(at, p.transform.position));
            }

            resources.Clear();
            seen.Clear();
            var hits = Physics.OverlapSphereNonAlloc(at, r, colliders, ResourceMask);
            if (hits > mostColliders) mostColliders = hits;
            for (var i = 0; i < hits; i++)
            {
                var res = Describe(colliders[i], out var pos);
                if (res != null) resources.Add(res, Vector3.Distance(at, pos));
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
            pc.Prefab = PrefabName(c.gameObject);
            if (c.GetLevel() > 1) pc.Stars = c.GetLevel() - 1;
            if (BaseAI.IsEnemy(c, me)) pc.Hostile = true;
            var ai = c.GetBaseAI();
            if (ai != null && ai.GetTargetCreature() == me) pc.TargetingMe = true;
            return pc;
        }

        /// <summary>The piece as 6.2 names it, or null for the walls, floors
        /// and everything else the bot has no use for.</summary>
        static PerceivedStructure Describe(Piece p)
        {
            var s = new PerceivedStructure { Prefab = PrefabName(p.gameObject), Pos = Convert.ToArray(p.transform.position) };
            if (p.TryGetComponent<Door>(out var door))
            {
                s.Kind = Wire.StructureDoor;
                var zdo = door.m_nview != null ? door.m_nview.GetZDO() : null;
                s.Open = zdo != null && zdo.GetInt(ZDOVars.s_state, 0) != 0;
            }
            else if (p.TryGetComponent<Bed>(out var bed))
            {
                s.Kind = Wire.StructureBed;
                if (bed.IsMine()) s.Mine = true;
            }
            else if (p.TryGetComponent<TeleportWorld>(out var portal))
            {
                s.Kind = Wire.StructurePortal;
                var tag = portal.GetText();
                if (!string.IsNullOrEmpty(tag)) s.Tag = tag;
            }
            else if (p.TryGetComponent<CraftingStation>(out _)) s.Kind = Wire.StructureStation;
            else if (p.TryGetComponent<Smelter>(out _)) s.Kind = Wire.StructureSmelter;
            else if (p.TryGetComponent<Fireplace>(out _)) s.Kind = Wire.StructureFire;
            else if (p.TryGetComponent<Container>(out _)) s.Kind = Wire.StructureContainer;
            else return null;
            s.Id = Id(p.m_nview);
            return s;
        }

        /// <summary>What a collider belongs to, if the bot could harvest or
        /// pick it up; each object once, however many colliders it has.</summary>
        PerceivedResource Describe(Collider col, out Vector3 pos)
        {
            pos = default;
            var nview = col.GetComponentInParent<ZNetView>();
            if (nview == null || !seen.Add(nview)) return null;
            var go = nview.gameObject;
            var r = new PerceivedResource { Prefab = PrefabName(go) };
            if (go.TryGetComponent<TreeBase>(out var tree))
            {
                r.Kind = Wire.ResourceTree;
                r.MinTool = tree.m_minToolTier;
            }
            else if (go.TryGetComponent<TreeLog>(out var log))
            {
                r.Kind = Wire.ResourceLog;
                r.MinTool = log.m_minToolTier;
            }
            else if (go.TryGetComponent<MineRock>(out var rock))
            {
                r.Kind = Wire.ResourceRock;
                r.MinTool = rock.m_minToolTier;
            }
            else if (go.TryGetComponent<MineRock5>(out var rock5))
            {
                r.Kind = Wire.ResourceRock;
                r.MinTool = rock5.m_minToolTier;
            }
            else if (go.TryGetComponent<Pickable>(out var pick))
            {
                if (pick.GetPicked()) return null;
                r.Kind = Wire.ResourcePickable;
            }
            else if (go.TryGetComponent<ItemDrop>(out var drop))
            {
                r.Kind = Wire.ResourceDrop;
                if (drop.m_itemData != null) r.Stack = drop.m_itemData.m_stack;
            }
            else return null;
            pos = go.transform.position;
            r.Pos = Convert.ToArray(pos);
            r.Id = Id(nview);
            return r;
        }

        static string Id(ZNetView nview)
        {
            var zdo = nview != null ? nview.GetZDO() : null;
            return zdo != null ? zdo.m_uid.ToString() : "";
        }

        static string PrefabName(GameObject go)
        {
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
