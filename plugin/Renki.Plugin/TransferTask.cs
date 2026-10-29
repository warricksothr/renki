using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.CompilerServices;
using HarmonyLib;
using Renki.Movement;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Use normal access/ownership RPCs, then the game's stack moves.
    /// Never create items, bypass wards, or mutate a remotely owned chest.</summary>
    sealed class TransferTask : IBodyTask
    {
        readonly TaskRequest request;
        readonly TransferArgs args;
        readonly Container chest;
        readonly bool withdraw;
        readonly Walker walker = new Walker();
        float requestedAt = -1;
        bool? granted;
        bool cancelled, opened;
        // A late response after cancellation must not open the inventory UI.
        // A lost RPC quarantines this chest until its reply arrives. Weak
        // keys let unloaded/destroyed chests and their tasks be collected.
        internal static readonly ConditionalWeakTable<Container, TransferTask> Pending = new ConditionalWeakTable<Container, TransferTask>();

        public TransferTask(TaskRequest request, TransferArgs args, Container chest)
        {
            this.request = request;
            this.args = args;
            this.chest = chest;
            withdraw = request.Task == TaskNames.Withdraw;
            walker.SetGoal(Player.m_localPlayer.transform.position, chest.transform.position);
            walker.ResetStuck(Player.m_localPlayer.transform.position);
        }

        public Outcome Tick(double dt)
        {
            var p = Player.m_localPlayer;
            if (p == null || p.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (chest == null || chest.m_nview == null || !chest.m_nview.IsValid()) return Finish(Outcome.Failed(Wire.ReasonTargetLost, "the chest is gone"));
            var pos = p.transform.position;
            if (!UseReach.Near(p, chest.gameObject, 2.5f))
            {
                if (requestedAt >= 0) return Finish(Outcome.Failed(Wire.ReasonUnreachable, "moved out of reach of the chest"));
                if (walker.Unreachable(pos, 2.5) is Outcome under) return Finish(under);
                if (Walker.Stranded(p) is Outcome stranded) return Finish(stranded);
                if (walker.Refused() is Outcome wet) return Finish(wet);
                if (!walker.Steer(pos, false)) return Finish(Outcome.Failed(Wire.ReasonStuck, "cannot reach the chest"));
                return null;
            }
            Driver.Stand();
            if (!PrivateArea.CheckAccess(chest.transform.position, 0f, false, false) ||
                !chest.CheckAccess(p.GetPlayerID())) return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "the chest is private or warded"));
            if (requestedAt < 0)
            {
                if (chest.IsInUse() || chest.m_nview.GetZDO().GetInt(ZDOVars.s_inUse) == 1)
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the chest is in use"));
                if (Pending.TryGetValue(chest, out _)) return Finish(Outcome.Failed(Wire.ReasonRefused, "still waiting for the previous chest request"));
                Pending.Add(chest, this);
                requestedAt = Time.unscaledTime;
                chest.Interact(p, false, false);
                return null;
            }
            if (granted == false) return Finish(Outcome.Failed(Wire.ReasonRefused, "the chest is in use or access was refused"));
            if (granted != true || !chest.IsOwner())
            {
                if (Time.unscaledTime - requestedAt > 5) return Finish(Outcome.Failed(Wire.ReasonTimeout, "the chest did not grant access"));
                return null;
            }
            if (!opened)
            {
                chest.Load(); // ownership was granted; read its current saved contents
                chest.SetInUse(true);
                opened = true;
                Report();
            }
            var pack = p.GetInventory();
            var source = withdraw ? chest.GetInventory() : pack;
            var destination = withdraw ? pack : chest.GetInventory();
            var result = Transfer.Run(args.Items, source.GetAllItems().ToArray(), Prefab,
                item => item.m_stack, item => item.m_equipped,
                (item, count) => Move(p, source, destination, item, count, withdraw));
            Report(result);
            RenkiPlugin.Instance.Send(InventoryReporter.Describe(pack));
            var moved = string.Join(", ", result.Moved.Select(kv => $"{kv.Value} {kv.Key}"));
            var left = string.Join(", ", result.Remaining.Select(kv => $"{kv.Value} {kv.Key}"));
            if (result.Remaining.Count > 0)
            {
                var missing = result.Remaining.Any(kv => source.GetAllItems().Where(it => Prefab(it) == kv.Key && !it.m_equipped).Sum(it => it.m_stack) < kv.Value);
                var heavy = withdraw && result.Remaining.Keys.Any(prefab => source.GetAllItems().Any(it => Prefab(it) == prefab && !it.m_equipped &&
                    it.m_stack > 0 && it.GetWeight() / it.m_stack > p.GetMaxCarryWeight() - pack.GetTotalWeight()));
                var reason = missing ? Wire.ReasonInvalidTarget : heavy ? Wire.ReasonTooHeavy : Wire.ReasonInventoryFull;
                var why = missing ? "requested items are missing or equipped" : heavy ? "carry weight limit" : "destination has no compatible stack or free slot";
                return Finish(Outcome.Failed(reason, $"moved {(moved.Length == 0 ? "nothing" : moved)}; could not move {left}: {why}"));
            }
            return Finish(Outcome.Succeeded(moved.Length == 0 ? "nothing to move" : $"moved {moved}"));
        }

        static string Prefab(ItemDrop.ItemData item) => item.m_dropPrefab != null ? item.m_dropPrefab.name : item.m_shared.m_name;

        static int Move(Player p, global::Inventory source, global::Inventory destination, ItemDrop.ItemData item, int wanted, bool weigh)
        {
            if (weigh)
            {
                var unit = item.m_stack > 0 ? item.GetWeight() / item.m_stack : 0;
                if (unit > 0) wanted = Math.Min(wanted, Math.Max(0, Mathf.FloorToInt((p.GetMaxCarryWeight() - destination.GetTotalWeight()) / unit)));
            }
            var before = item.m_stack;
            var left = wanted;
            // Use IsSameType, including quality/world-level metadata, rather
            // than assuming that equal prefab names can always be stacked.
            foreach (var stack in destination.GetAllItems().ToArray())
            {
                if (left == 0) break;
                if (!stack.IsSameType(item)) continue;
                var n = Math.Min(left, stack.m_shared.m_maxStackSize - stack.m_stack);
                if (n <= 0) continue;
                var was = item.m_stack;
                destination.MoveItemToThis(source, item, n, stack.m_gridPos.x, stack.m_gridPos.y);
                left -= was - item.m_stack;
            }
            while (left > 0)
            {
                var slot = destination.FindEmptySlot(true);
                if (slot.x < 0) break;
                var was = item.m_stack;
                destination.MoveItemToThis(source, item, Math.Min(left, item.m_shared.m_maxStackSize), slot.x, slot.y);
                var n = was - item.m_stack;
                if (n == 0) break;
                left -= n;
            }
            return before - item.m_stack;
        }

        void Report(TransferResult result = null)
        {
            var inv = InventoryReporter.Describe(chest.GetInventory());
            RenkiPlugin.Instance.Send(new ContainerSnapshot
            {
                Entity = args.Container,
                Pos = Convert.ToArray(chest.transform.position),
                Size = inv.Size,
                Items = inv.Items,
                Task = result != null ? request.Id : null,
                Direction = result != null ? request.Task : null,
                Moved = result?.Moved.Count > 0 ? result.Moved : null,
                Remaining = result?.Remaining.Count > 0 ? result.Remaining : null,
            });
        }

        Outcome Finish(Outcome result) { Cancel(); return result; }

        public void Cancel()
        {
            cancelled = true;
            Driver.Stand();
            if (chest != null && opened && chest.IsOwner()) chest.SetInUse(false);
            opened = false;
        }

        internal void Response(bool allowed)
        {
            granted = allowed;
            // Cancelled requests release a granted chest without ever using it.
            if (cancelled && allowed && chest != null && chest.IsOwner()) chest.SetInUse(false);
        }
    }

    [HarmonyPatch(typeof(Container), "RPC_OpenResponse")]
    static class TransferOpenResponse
    {
        static bool Prefix(Container __instance, bool granted)
        {
            if (!TransferTask.Pending.TryGetValue(__instance, out var task)) return true;
            TransferTask.Pending.Remove(__instance);
            task.Response(granted);
            return false;
        }
    }
}
