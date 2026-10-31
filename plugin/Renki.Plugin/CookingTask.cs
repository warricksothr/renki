using System;
using System.Collections.Generic;
using System.Linq;
using HarmonyLib;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One normally inserted ingredient, a verified owned slot, normal
    /// removal and provenance-linked dropped output. Never edits inventory/ZDOs.</summary>
    sealed partial class CookingTask : IBodyTask
    {
        internal static CookingTask Active;
        readonly string id, raw;
        readonly CookingStation station;
        readonly MoveToTask approach;
        readonly CookingFire fire;
        readonly List<Character> creatures = new List<Character>();
        readonly List<ItemDrop> outputs = new List<ItemDrop>();
        CookingBatch batch;
        PickupTask pickup;
        bool arrived, removing, cancelled;
        float began, previousHp;
        int spawned, received;
        string lastPhase;
        public string Output { get; }

        internal static bool Supported(CookingStation s) => s != null && s.m_requireFire && !s.m_useFuel &&
            s.m_addFoodSwitch == null && s.m_slots != null && s.m_slots.Length > 0 && s.m_slots.Length <= 16;

        internal static bool Heated(CookingStation s) => s.m_requireFire && s.IsFireLit() ||
            s.m_useFuel && s.GetFuel() > 0 && (s.m_useFueldWhileEmpty || s.HaveUncookedItem());

        internal static List<CookSlot> Slots(CookingStation s)
        {
            var zdo = s.m_nview?.GetZDO();
            if (zdo == null || s.m_slots == null || s.m_slots.Length > 16) return null;
            var slots = new List<CookSlot>(s.m_slots.Length);
            for (var i = 0; i < s.m_slots.Length; i++)
                slots.Add(new CookSlot
                {
                    Item = zdo.GetString("slot" + i),
                    Status = zdo.GetInt("slotstatus" + i),
                    Producer = zdo.GetLong(ZDOVars.s_playerID + i)
                });
            return slots;
        }

        internal static CookingStatus Observe(CookingStation s)
        {
            var slots = Slots(s);
            if (slots == null) return null;
            var result = new CookingStatus { Heated = Heated(s), Supported = Supported(s) };
            for (var i = 0; i < slots.Count; i++)
                result.Slots.Add(new CookingSlotStatus
                {
                    Index = i,
                    Item = slots[i].Item,
                    State = string.IsNullOrEmpty(slots[i].Item) ? "empty" : slots[i].Status == 0 ? "cooking"
                        : slots[i].Status == 1 ? "done" : slots[i].Status == 2 ? "burnt" : "unknown",
                    ElapsedS = Math.Round(s.m_nview.GetZDO().GetFloat("slot" + i), 1)
                });
            return result;
        }

        public CookingTask(string id, InteractArgs args, CookingStation station)
        {
            if (!Supported(station) || !CookingBatch.Supported(args.Cook))
                throw new TaskRefused(Wire.ReasonInvalidTarget, "cooking currently supports basic fire-backed meat/neck-tail stations");
            this.id = id; raw = args.Cook; Output = CookingBatch.Result(raw); this.station = station;
            fire = new CookingFire(station, id);
            var pos = station.transform.position;
            approach = new MoveToTask(new MoveToArgs
            {
                To = new Target { Pos = Convert.ToV3(pos) },
                StopDistance = 2,
                Dry = true,
                AvoidHostiles = true
            }, pos, null);
            previousHp = Player.m_localPlayer.GetHealth();
            Active = this;
        }

        static int Count(Player me, string prefab) => me.GetInventory().GetAllItems()
            .Where(i => i.m_dropPrefab != null && i.m_dropPrefab.name == prefab).Sum(i => i.m_stack);

        internal static float InteractionRange(Player me) => float.IsFinite(me.m_maxInteractDistance) && me.m_maxInteractDistance > 0
            ? me.m_maxInteractDistance : 0;

        public Outcome Tick(double dt)
        {
            var me = Player.m_localPlayer;
            if (cancelled || me == null || me.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "cooking stopped"));
            if (station == null || station.m_nview == null || !station.m_nview.IsValid())
                return Finish(Outcome.Failed(Wire.ReasonTargetLost, "the cooking station is gone"));
            if (!PrivateArea.CheckAccess(station.transform.position, 0f, false, false) || me.NoCostCheat())
                return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "normal cooking access is required"));
            var hp = me.GetHealth();
            if (hp < Mathf.Min(previousHp, me.GetMaxHealth()) - .15f || hp < me.GetMaxHealth() * .8f ||
                me.GetFoods().Count == 0 || me.InWater() || me.IsSwimming())
                return Finish(Outcome.Failed(Wire.ReasonRefused, "unsafe health/food/water for cooking; any inserted input remains on the station"));
            previousHp = hp;
            creatures.Clear(); Character.GetCharactersInRange(me.transform.position, 12, creatures);
            if (creatures.Any(c => c != me && !c.IsDead() && !(c is Player) && !c.IsTamed() &&
                !(c.GetBaseAI() is AnimalAI) && BaseAI.IsEnemy(me, c)))
                return Finish(Outcome.Failed(Wire.ReasonRefused, "a hostile is too close to tend food"));
            if (!arrived)
            {
                if (!UseReach.Near(me, station.gameObject, InteractionRange(me)))
                {
                    var result = approach.Tick(dt);
                    if (result == null) return null;
                    if (result.Status != Wire.StatusSucceeded) return Finish(result);
                }
                arrived = true;
            }
            if (batch == null)
            {
                var currentSlots = Slots(station);
                if (currentSlots == null || currentSlots.Any(s => !string.IsNullOrEmpty(s.Item)))
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the cooking station is busy or unavailable; not fueling another batch"));
                if (InventoryGui.instance == null)
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the normal game inventory interface is unavailable"));
                var conversion = station.GetItemConversion(raw);
                var ingredient = me.GetInventory().GetAllItems().FirstOrDefault(i => i.m_dropPrefab != null && i.m_dropPrefab.name == raw);
                if (ingredient == null || conversion?.m_to == null || conversion.m_to.name != Output ||
                    conversion.m_to.m_itemData.m_shared.m_food <= 0 || !float.IsFinite(conversion.m_cookTime) ||
                    conversion.m_cookTime < 1 || conversion.m_cookTime > 120)
                    return Finish(Outcome.Failed(Wire.ReasonInvalidTarget, $"carry {raw} and use a station with its ordinary cooking recipe"));
                var maxYield = 1 + (station.m_canGiveBonusYield && InventoryGui.instance != null ? Math.Max(0, InventoryGui.instance.m_craftBonusAmount) : 0);
                if (maxYield > 10 || !me.GetInventory().CanAddItem(conversion.m_to.m_itemData, maxYield) ||
                    me.GetInventory().GetTotalWeight() + conversion.m_to.m_itemData.GetWeight() * maxYield > me.GetMaxCarryWeight())
                    return Finish(Outcome.Failed(Wire.ReasonInventoryFull, "reserve space and carry weight for the full possible cooking yield"));
                if (!UseReach.Near(me, station.gameObject, InteractionRange(me)))
                    return Finish(Outcome.Failed(Wire.ReasonUnreachable, "the cooking station is not in sight/reach"));
                var fireResult = fire.Tick(me, out var warming);
                if (fireResult != null) return Finish(fireResult);
                if (warming) return null;
                batch = new CookingBatch(raw, ZNet.GetUID(), Count(me, raw), Count(me, Output), maxYield, conversion.m_cookTime);
                began = Time.unscaledTime;
            }
            ObserveRemoteOutputs();
            if (deliveryFailure != null) return Finish(Outcome.Failed(Wire.ReasonRefused, deliveryFailure));
            var action = batch.Next(Time.unscaledTime - began, Count(me, raw), Count(me, Output), Slots(station), Heated(station), spawned, received,
                deliverySettled: remoteOutput == null || remoteOutput.Complete && Time.unscaledTime - removalAt >= 3);
            if (batch.Phase != lastPhase)
            {
                lastPhase = batch.Phase;
                Trace("phase changed", new Dictionary<string, object>
                {
                    ["phase"] = lastPhase,
                    ["slot"] = batch.Slot,
                    ["reason"] = batch.Failure
                });
            }
            if (action == CookAction.Failed) return Finish(Outcome.Failed(Wire.ReasonRefused, batch.Failure));
            if (action == CookAction.Succeeded)
            {
                Trace("verified cooked inventory receipt", new Dictionary<string, object>
                { ["output"] = Output, ["count"] = batch.OutputCount, ["evidence"] = remoteOutput == null ? "direct_spawn" : "remote_correlation" });
                return Finish(Outcome.Succeeded($"cooked one {raw}; collected {batch.OutputCount} {Output}"));
            }
            if (action == CookAction.Collect)
            {
                if (pickup == null)
                {
                    var drop = outputs.FirstOrDefault(d => d != null);
                    if (drop != null) pickup = new PickupTask(drop, safeTravel: true, allowMerge: false);
                }
                if (pickup != null)
                {
                    var result = pickup.Tick(dt);
                    if (result != null)
                    {
                        pickup.Cancel(); pickup = null;
                        if (result.Status != Wire.StatusSucceeded) return Finish(result);
                    }
                }
                return null;
            }
            Driver.Stand(); Driver.LookAt = station.transform.position; Driver.LookUntil = Time.time + 1;
            if (!UseReach.Near(me, station.gameObject, InteractionRange(me)))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "the cooking station is no longer in sight/reach"));
            if (action == CookAction.Insert)
            {
                var item = me.GetInventory().GetAllItems().FirstOrDefault(i => i.m_dropPrefab != null && i.m_dropPrefab.name == raw);
                if (item == null || !station.UseItem(me, item))
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the station refused the carried ingredient"));
                Trace("insert requested; awaiting slot and inventory receipt");
            }
            if (action == CookAction.Remove)
            {
                removing = true;
                bool used;
                issuingRemoval = true;
                try { used = station.Interact(me, false, false); }
                finally { issuingRemoval = false; }
                if (!used)
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the station refused output removal"));
                if (!removalSeen)
                    return Finish(Outcome.Failed(Wire.ReasonRefused, "the normal cooking removal request was not observed"));
                Trace("cooked output removal requested");
            }
            return null;
        }

        void Trace(string message, Dictionary<string, object> data = null)
        {
            data ??= new Dictionary<string, object>(); data["id"] = id; data["input"] = raw;
            data["station"] = Perceiver.Id(station.m_nview);
            Tracer.Emit("cooking", message, data);
        }

        Outcome Finish(Outcome outcome) { Cancel(); return outcome; }
        public void Cancel()
        {
            if (cancelled) return;
            cancelled = true; approach.Cancel(); pickup?.Cancel(); Driver.Stand();
            if (Active == this) Active = null;
        }

        [HarmonyPatch(typeof(CookingStation), "SpawnItem")]
        static class OutputReceipt
        {
            static void Prefix(CookingStation __instance, string name, int slot, out HashSet<ItemDrop> __state)
            {
                var task = Active;
                __state = task != null && task.station == __instance && task.removing &&
                    task.batch?.Slot == slot && task.Output == name ? new HashSet<ItemDrop>(ItemDrop.s_instances) : null;
            }
            static void Postfix(HashSet<ItemDrop> __state)
            {
                var task = Active;
                if (__state == null || task == null) return;
                foreach (var drop in ItemDrop.s_instances)
                    if (drop != null && !__state.Contains(drop) && drop.m_itemData?.m_dropPrefab?.name == task.Output)
                    {
                        task.AddOutput(drop, "normal station output spawned");
                    }
            }
        }

        sealed class PickupAttempt
        {
            public CookingTask Task;
            public string Entity;
            public int Before, Stack;
        }

        [HarmonyPatch(typeof(Humanoid), "Pickup")]
        static class PickupReceipt
        {
            static void Prefix(Humanoid __instance, GameObject go, out PickupAttempt __state)
            {
                __state = null;
                var task = Active;
                if (task == null || __instance != Player.m_localPlayer || go == null) return;
                var drop = go.GetComponent<ItemDrop>();
                task.ObserveRemoteOutput(drop);
                if (drop == null || !task.outputs.Contains(drop)) return;
                __state = new PickupAttempt
                {
                    Task = task,
                    Entity = Perceiver.Id(drop.m_nview),
                    Before = Count(Player.m_localPlayer, task.Output),
                    Stack = drop.m_itemData.m_stack
                };
            }
            static void Postfix(PickupAttempt __state)
            {
                if (__state == null || Active != __state.Task || Player.m_localPlayer == null) return;
                var task = __state.Task;
                var gained = Count(Player.m_localPlayer, task.Output) - __state.Before;
                if (gained <= 0 || gained > __state.Stack) return;
                task.received += gained;
                task.Trace("verified cooking drop pickup", new Dictionary<string, object>
                { ["entity"] = __state.Entity, ["output"] = task.Output, ["count"] = gained });
            }
        }
    }
}
