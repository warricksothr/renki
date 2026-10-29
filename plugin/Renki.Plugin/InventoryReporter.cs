using System.Collections.Generic;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// What the bot carries (docs/BRIDGE-PROTOCOL.md 6.3): sent once after
    /// welcome and whenever the inventory changes. The game says when it
    /// changes (Inventory.m_onChanged); a burst of changes, such as picking
    /// up a pile, goes out as one message.
    /// </summary>
    sealed class InventoryReporter
    {
        /// <summary>The least time between two messages, so a burst is one.</summary>
        const float Gap = 0.25f;

        /// <summary>How often worn and equipped items are checked: wear, and
        /// perhaps equipping, do not fire m_onChanged, so a swinging axe or
        /// a newly drawn one would otherwise go stale.</summary>
        const float WearEvery = 2f;

        readonly RenkiPlugin plugin;
        Inventory watched;
        bool dirty = true;
        float next;
        float nextWear;
        double sentWear;
        int sentEquipped;

        public InventoryReporter(RenkiPlugin plugin)
        {
            this.plugin = plugin;
        }

        /// <summary>Send it again on the next Update, as after welcome.</summary>
        public void Resend() => dirty = true;

        public void Update(bool inSession)
        {
            var p = Player.m_localPlayer;
            var inv = p != null ? p.GetInventory() : null;
            if (inv != watched)
            {
                // A new player after a respawn has a new inventory.
                if (watched != null) watched.m_onChanged -= Changed;
                watched = inv;
                if (inv != null) inv.m_onChanged += Changed;
                dirty = true;
            }
            if (inv != null && Time.unscaledTime >= nextWear)
            {
                nextWear = Time.unscaledTime + WearEvery;
                // Against what was last sent, so slow wear adds up.
                if (System.Math.Abs(Wear(inv) - sentWear) >= 1 || Equipped(inv) != sentEquipped) dirty = true;
            }
            if (!inSession || inv == null || !dirty || Time.unscaledTime < next) return;
            dirty = false;
            next = Time.unscaledTime + Gap;
            sentWear = Wear(inv);
            sentEquipped = Equipped(inv);
            plugin.Send(Describe(inv));
        }

        void Changed() => dirty = true;

        /// <summary>Which slots hold equipped items, as one number.</summary>
        static int Equipped(global::Inventory inv)
        {
            var key = 17;
            foreach (var item in inv.GetAllItems())
                if (item.m_equipped) key = key * 31 + item.m_gridPos.x * 64 + item.m_gridPos.y;
            return key;
        }

        static double Wear(global::Inventory inv)
        {
            var wear = 0.0;
            foreach (var item in inv.GetAllItems())
                if (item.m_shared.m_useDurability) wear += item.m_durability;
            return wear;
        }

        static Protocol.Inventory Describe(global::Inventory inv)
        {
            var m = new Protocol.Inventory { Size = new[] { inv.GetWidth(), inv.GetHeight() } };
            foreach (var item in inv.GetAllItems())
            {
                var shared = item.m_shared;
                var seen = new InventoryItem
                {
                    Slot = new[] { item.m_gridPos.x, item.m_gridPos.y },
                    // The few items with no prefab go by their name token (6.3).
                    Prefab = item.m_dropPrefab != null ? item.m_dropPrefab.name : shared.m_name,
                    Stack = item.m_stack,
                    // One of the stack, so the controller can weigh what to drop.
                    Weight = System.Math.Round(shared.m_weight, 2),
                    Type = shared.m_itemType.ToString(),
                };
                if (shared.m_value > 0) seen.Value = shared.m_value;
                if (shared.m_maxStackSize > 1) seen.StackMax = shared.m_maxStackSize;
                if (shared.m_maxQuality > 1) seen.Quality = item.m_quality;
                if (shared.m_useDurability)
                {
                    seen.Durability = System.Math.Round(item.m_durability, 1);
                    seen.DurabilityMax = System.Math.Round(item.GetMaxDurability(), 1);
                }
                if (item.m_equipped) seen.Equipped = true;
                m.Items.Add(seen);
            }
            m.Items.Sort((a, b) => a.Slot[1] != b.Slot[1] ? a.Slot[1].CompareTo(b.Slot[1]) : a.Slot[0].CompareTo(b.Slot[0]));
            return m;
        }
    }
}
