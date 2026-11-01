using System.Linq;
using HarmonyLib;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    // Observe both normal auto-pickup and explicit pickup, never chest withdrawal.
    [HarmonyPatch(typeof(Humanoid), nameof(Humanoid.Pickup))]
    static class PickupObservation
    {
        sealed class Before
        {
            internal string Prefab, Entity, World, WorldUid;
            internal double[] Pos;
            internal int Count;
            internal Player Player;
        }

        static int Count(Player player, string prefab) => player.GetInventory().GetAllItems()
            .Where(item => item.m_dropPrefab != null && item.m_dropPrefab.name == prefab).Sum(item => item.m_stack);

        static void Prefix(Humanoid __instance, GameObject go, out Before __state)
        {
            __state = null;
            var player = Player.m_localPlayer;
            if (RenkiPlugin.Instance?.ObservePickups != true || player == null || __instance != player || go == null || ZNet.instance == null) return;
            var drop = go.GetComponent<ItemDrop>();
            if (drop == null || drop.m_itemData?.m_dropPrefab == null) return;
            var prefab = drop.m_itemData.m_dropPrefab.name;
            __state = new Before
            {
                World = ZNet.instance.GetWorldName(),
                WorldUid = ZNet.instance.GetWorldUID().ToString(System.Globalization.CultureInfo.InvariantCulture),
                Player = player,
                Prefab = prefab,
                Entity = Perceiver.Id(drop.m_nview),
                Pos = Convert.ToArray(go.transform.position),
                Count = Count(player, prefab)
            };
        }

        static void Postfix(Before __state)
        {
            // A partially fitting pickup may return false after adding a stack.
            // The observed positive pack delta is the authority for its count.
            if (__state == null || __state.Player == null || __state.Player != Player.m_localPlayer ||
                RenkiPlugin.Instance?.ObservePickups != true) return;
            if (ZNet.instance == null || ZNet.instance.GetWorldName() != __state.World ||
                ZNet.instance.GetWorldUID().ToString(System.Globalization.CultureInfo.InvariantCulture) != __state.WorldUid) return;
            var gained = Count(__state.Player, __state.Prefab) - __state.Count;
            if (gained > 0) RenkiPlugin.Instance.Send(new Renki.Protocol.Event
            {
                Name = Wire.EventItemPickedUp,
                World = __state.World,
                WorldUid = __state.WorldUid,
                Prefab = __state.Prefab,
                Count = gained,
                By = __state.Entity,
                Pos = __state.Pos
            });
        }
    }
}
