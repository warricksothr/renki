using System;
using System.Collections.Generic;
using System.Linq;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>Normal station use followed by the inventory repair button.
    /// The game decides roof, level, recipe and repair eligibility. No items
    /// are created and this routine never writes durability itself.</summary>
    sealed class RepairTask : IBodyTask
    {
        readonly CraftingStation station;
        readonly MoveToTask approach;
        bool arrived, opened;
        int repaired;
        float began, next;

        public RepairTask(CraftingStation station)
        {
            this.station = station;
            var pos = station.transform.position;
            approach = new MoveToTask(new MoveToArgs { To = new Target { Pos = Convert.ToV3(pos) }, StopDistance = 2, Dry = true }, pos, null);
        }

        public Outcome Tick(double dt)
        {
            var me = Player.m_localPlayer;
            if (me == null || me.IsDead()) return Finish(Outcome.Failed(Wire.ReasonStopped, "not in the world"));
            if (station == null) return Finish(Outcome.Failed(Wire.ReasonTargetLost, "the station is gone"));
            if (!PrivateArea.CheckAccess(station.transform.position, 0f, false, false))
                return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "the station is behind a ward"));
            if (me.NoCostCheat()) return Finish(Outcome.Failed(Wire.ReasonNotAllowed, "repair requires normal station rules"));
            if (!arrived)
            {
                var result = approach.Tick(dt);
                if (result == null) return null;
                if (result.Status != Wire.StatusSucceeded) return Finish(result);
                arrived = true;
            }
            Driver.Stand();
            Driver.LookAt = station.transform.position; Driver.LookUntil = Time.time + 1;
            if (!UseReach.Near(me, station.gameObject, 2.5f))
                return Finish(Outcome.Failed(Wire.ReasonUnreachable, "the station is not in interaction sight/reach"));
            if (!opened)
            {
                // CraftingStation.Interact returns false even when it opens.
                // Confirm the player's actual station/UI, as with bed use.
                var heard = new List<string>();
                using (InteractTask.Listen(heard)) station.Interact(me, false, false);
                opened = true; began = Time.unscaledTime;
                if (me.GetCurrentCraftingStation() != station || !InventoryGui.IsVisible())
                    return Finish(Outcome.Failed(Wire.ReasonRefused, heard.Count > 0 ? string.Join("; ", heard) : "the station refused use"));
                return null;
            }
            if (me.GetCurrentCraftingStation() != station || InventoryGui.instance == null ||
                !station.CheckUsable(me, false) || Vector3.Distance(me.transform.position, station.transform.position) > 3)
                return Finish(Outcome.Failed(Wire.ReasonRefused, "the station is not usable for repair"));
            var gui = InventoryGui.instance;
            if (!gui.HaveRepairableItems())
                return Finish(Outcome.Succeeded($"repaired {repaired} carried item(s); no more gear repairable at this station"));
            if (repaired >= 32 || Time.unscaledTime - began > 10)
                return Finish(Outcome.Failed(Wire.ReasonRefused, "bounded station repair limit reached"));
            if (Time.unscaledTime < next) return null;
            next = Time.unscaledTime + .25f;
            var before = me.GetInventory().GetAllItems().ToDictionary(item => item, item => item.m_durability);
            // This is the same method invoked by the normal repair button;
            // CanRepair and CheckUsable remain the game's authority.
            gui.RepairOneItem();
            var changed = before.Where(pair => pair.Key.m_durability > pair.Value).ToArray();
            if (changed.Length != 1 || !float.IsFinite(changed[0].Key.m_durability) ||
                changed[0].Key.m_durability > changed[0].Key.GetMaxDurability())
                return Finish(Outcome.Failed(Wire.ReasonRefused, "repair button produced no single verified durability gain"));
            var item = changed[0].Key;
            repaired++;
            Tracer.Emit("repair", "verified item repair", new Dictionary<string, object>
            {
                ["prefab"] = item.m_dropPrefab != null ? item.m_dropPrefab.name : null,
                ["slot"] = new[] { item.m_gridPos.x, item.m_gridPos.y },
                ["before"] = changed[0].Value,
                ["after"] = item.m_durability,
                ["max"] = item.GetMaxDurability(),
                ["station"] = Convert.ToArray(station.transform.position),
            });
            return null;
        }

        Outcome Finish(Outcome outcome) { Cancel(); return outcome; }
        public void Cancel()
        {
            approach.Cancel(); Driver.Stand();
            var me = Player.m_localPlayer;
            if (opened && me != null && me.GetCurrentCraftingStation() == station)
            {
                if (InventoryGui.instance != null) InventoryGui.instance.Hide();
                me.SetCraftingStation(null);
            }
        }
    }
}
