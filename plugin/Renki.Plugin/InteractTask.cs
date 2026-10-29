using System.Collections.Generic;
using HarmonyLib;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// interact: walk to a thing and use it as a player would, through its
    /// Interactable (docs/BRIDGE-PROTOCOL.md 7.2). The game answers true or
    /// false, and says why on the screen; the task reports both, so a ward, a
    /// bed someone else owns or a missing roof comes back as the failure
    /// <c>refused</c> with the game's own words.
    /// </summary>
    sealed class InteractTask : IBodyTask
    {
        /// <summary>How near the thing's surface the bot uses it from. The
        /// game allows 5 m; nearer, the bot is plainly at it.</summary>
        const float Reach = 2.0f;

        /// <summary>How long a hold keeps the use key down.</summary>
        const float HoldFor = 1.0f;

        /// <summary>How often a held key repeats: a little slower than the
        /// game's 0.2 s, which it would turn away.</summary>
        const float Repeat = 0.25f;

        readonly InteractArgs args;
        readonly Interactable target;
        readonly GameObject obj;
        readonly string name;
        readonly Walker walker = new Walker();
        Vector3 goal;
        float holdUntil = -1, nextRepeat;
        bool used;
        readonly List<string> heard = new List<string>();

        public InteractTask(InteractArgs args, Interactable target)
        {
            this.args = args;
            this.target = target;
            obj = ((MonoBehaviour)target).gameObject;
            name = Name(target);
            var p = Player.m_localPlayer;
            goal = obj.transform.position;
            walker.SetGoal(p.transform.position, goal);
            walker.ResetStuck(p.transform.position);
        }

        /// <summary>The kind InteractRule decides by, from the game's type.</summary>
        public static string Kind(Interactable it)
        {
            var go = ((MonoBehaviour)it).gameObject;
            switch (it)
            {
                // Its own grave it empties into the pack when all of it
                // fits; otherwise the game opens it as a chest.
                case TombStone t:
                    return t.IsOwner() && t.EasyFitInInventory(Player.m_localPlayer) ? InteractRule.Other : InteractRule.Container;
                case OfferingBowl _: return InteractRule.Boss;
                case ShipControlls _: return InteractRule.Ship;
                case Pickable _: return InteractRule.Pickable;
                case ItemDrop _: return InteractRule.Drop;
                case Container _: return InteractRule.Container;
                case CraftingStation _: return InteractRule.Station;
                case Sign _:
                case TeleportWorld _: return InteractRule.Text;
                case PrivateArea _: return InteractRule.Ward;
            }
            // A station's parts can carry their own Interactable.
            return go.GetComponentInParent<CraftingStation>() != null ? InteractRule.Station : InteractRule.Other;
        }

        public static string Name(Interactable it) =>
            it is Hoverable h ? Localize(h.GetHoverName()) : ((MonoBehaviour)it).gameObject.name;

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null) return Outcome.Failed(Wire.ReasonStopped, "no player");
            // Used up by the first press of a hold is still used.
            if (holdUntil >= 0) return obj == null ? Answer(used) : Holding(p);
            if (obj == null) return Outcome.Failed(Wire.ReasonTargetLost, $"the {name} is gone");
            var pos = p.transform.position;
            if (Near(p))
            {
                Driver.Stand();
                walker.ResetStuck(pos);
                Driver.LookAt = obj.transform.position;
                Driver.LookUntil = Time.time + 1f;
                if (p.InAttack() || p.InDodge()) return null;
                // The pack may have filled on the way, and then the game
                // would open the grave as a chest.
                if (target is TombStone && Kind(target) != InteractRule.Other)
                    return Outcome.Failed(Wire.ReasonInventoryFull, $"not all of the {name} fits in the pack");
                var ok = Use(p, false);
                if (!args.Hold) return Answer(ok);
                used = ok;
                holdUntil = Time.time + HoldFor;
                nextRepeat = Time.time + Repeat;
                return null;
            }
            if ((obj.transform.position - goal).sqrMagnitude > 1)
            {
                goal = obj.transform.position;
                walker.SetGoal(pos, goal);
            }
            if (walker.Unreachable(pos, Reach) is Outcome under)
            {
                Driver.Stand();
                return under;
            }
            if (Walker.Stranded(p) is Outcome stranded)
            {
                Driver.Stand();
                return stranded;
            }
            if (walker.Refused() is Outcome wet)
            {
                Driver.Stand();
                return wet;
            }
            if (!walker.Steer(pos, false))
            {
                Driver.Stand();
                return Outcome.Failed(Wire.ReasonStuck,
                    $"no progress after {walker.RecoveryAttempts} recoveries{(walker.StraightLine ? " (no navmesh path)" : "")}");
            }
            return null;
        }

        Outcome Holding(Player p)
        {
            if (Time.time >= holdUntil) return Answer(used);
            if (Time.time >= nextRepeat)
            {
                nextRepeat = Time.time + Repeat;
                used |= Use(p, true);
            }
            return null;
        }

        /// <summary>Within reach of the thing's surface, on its level.</summary>
        bool Near(Player p)
        {
            var chest = p.transform.position + Vector3.up;
            var best = float.MaxValue;
            foreach (var c in obj.GetComponentsInChildren<Collider>())
            {
                if (!c.enabled || c.isTrigger) continue;
                // Unity gives a concave mesh's ClosestPoint as the point
                // itself; its bounds are the nearest it can say.
                var at = c is MeshCollider m && !m.convex ? c.bounds.ClosestPoint(chest) : c.ClosestPoint(chest);
                best = Mathf.Min(best, Vector3.Distance(chest, at));
            }
            if (best < float.MaxValue) return best <= Reach;
            return Arrival.Reached(Convert.ToV3(p.transform.position), Convert.ToV3(obj.transform.position), Reach);
        }

        /// <summary>One press of the use key, hearing what the game says.</summary>
        bool Use(Player p, bool hold)
        {
            Heard.Into = heard;
            try
            {
                if (!target.Interact(p, hold, args.Alt)) return false;
                p.DoInteractAnimation(obj);
                return true;
            }
            finally
            {
                Heard.Into = null;
                // Whatever opened a window anyway (a tombstone too full to
                // take at once opens as a chest) is shut: nobody is there to use it.
                if (InventoryGui.IsVisible()) InventoryGui.instance.Hide();
            }
        }

        Outcome Answer(bool ok)
        {
            var said = heard.Count > 0 ? string.Join("; ", heard) : null;
            Tracer.Emit("interact", ok ? "used" : "refused", new Dictionary<string, object>
            {
                ["target"] = name,
                ["alt"] = args.Alt,
                ["hold"] = args.Hold,
                ["said"] = said,
            });
            if (ok) return Outcome.Succeeded(said != null ? $"used the {name}: {said}" : $"used the {name}");
            return Outcome.Failed(Wire.ReasonRefused, said ?? $"the {name} did nothing");
        }

        public void Cancel() => Driver.Stand();

        static string Localize(string s) => Localization.instance != null ? Localization.instance.Localize(s) : s;

        /// <summary>What the game says to the bot while it uses something:
        /// the line a player would see in the middle of the screen.</summary>
        [HarmonyPatch(typeof(Player), nameof(Player.Message))]
        static class Heard
        {
            public static List<string> Into;

            static void Postfix(Player __instance, string msg)
            {
                if (Into == null || __instance != Player.m_localPlayer || string.IsNullOrEmpty(msg)) return;
                // A held key hears the same line on every repeat.
                var line = Localize(msg).Trim();
                if (!Into.Contains(line)) Into.Add(line);
            }
        }
    }
}
