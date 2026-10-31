using System;
using System.Collections.Generic;
using System.Linq;
using HarmonyLib;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    sealed partial class CookingTask
    {
        CookingRemoteOutput remoteOutput;
        bool issuingRemoval, removalSeen;
        float removalAt;
        long removalTicks;
        Vector3 outputOrigin;
        string deliveryFailure;
        readonly HashSet<string> inspectedOutputs = new HashSet<string>();

        void AddOutput(ItemDrop drop, string evidence)
        {
            if (outputs.Contains(drop)) return;
            outputs.Add(drop); spawned += drop.m_itemData.m_stack;
            Trace(evidence, new Dictionary<string, object>
            { ["entity"] = Perceiver.Id(drop.m_nview), ["output"] = Output, ["count"] = drop.m_itemData.m_stack });
        }

        void ObserveRemoteOutput(ItemDrop drop)
        {
            if (remoteOutput == null || drop == null || drop.m_nview == null || !drop.m_nview.IsValid()) return;
            var zdo = drop.m_nview.GetZDO();
            var entity = Perceiver.Id(drop.m_nview);
            var born = (zdo.GetLong(ZDOVars.s_spawnTime) - removalTicks) / (double)TimeSpan.TicksPerSecond;
            var distance = Vector3.Distance(drop.transform.position, outputOrigin);
            var result = remoteOutput.Observe(entity, drop.m_itemData.m_dropPrefab?.name,
                zdo.m_uid.UserID, born, distance, drop.m_itemData.m_stack, drop.m_itemData.m_crafterID);
            if (result == CookingDropMatch.Ambiguous) deliveryFailure = remoteOutput.Failure;
            if (result == CookingDropMatch.Accepted) AddOutput(drop, "correlated remote station output observed");
            if (result == CookingDropMatch.Ignore && !outputs.Contains(drop) && drop.m_itemData.m_dropPrefab?.name == Output &&
                distance <= 6 && inspectedOutputs.Count < 8 && inspectedOutputs.Add(entity))
                Trace("remote output candidate ignored", new Dictionary<string, object>
                {
                    ["entity"] = entity,
                    ["creator"] = zdo.m_uid.UserID,
                    ["bornAfterRequestS"] = born,
                    ["distance"] = distance,
                    ["stack"] = drop.m_itemData.m_stack,
                    ["crafter"] = drop.m_itemData.m_crafterID
                });
        }

        void ObserveRemoteOutputs()
        {
            if (remoteOutput == null) return;
            foreach (var drop in ItemDrop.s_instances) ObserveRemoteOutput(drop);
            if (outputs.Any(d => d != null && d.m_itemData.m_stack != 1))
                deliveryFailure = "a correlated cooking drop merged or changed before collection";
        }

        // Observe only the RPC synchronously issued by this task's normal use.
        // No packet is modified or resent, and remote simulation stays remote.
        [HarmonyPatch(typeof(ZNetView), nameof(ZNetView.InvokeRPC), new Type[] { typeof(string), typeof(object[]) })]
        static class RemovalRequestReceipt
        {
            static void Prefix(ZNetView __instance, string method, object[] parameters)
            {
                var task = Active;
                if (task == null || !task.issuingRemoval || __instance != task.station.m_nview || method != "RPC_RemoveDoneItem") return;
                if (parameters.Length != 2 || !(parameters[0] is Vector3 userPoint) || !(parameters[1] is int count) || !task.batch.ExpectYield(count))
                { task.deliveryFailure = "normal cooking removal request was not verified"; return; }
                task.removalSeen = true;
                var owner = __instance.GetZDO().GetOwner();
                task.removalAt = Time.unscaledTime;
                task.removalTicks = ZNet.instance.GetTime().Ticks;
                var station = task.station;
                var slot = station.m_slots[task.batch.Slot].position;
                var direction = userPoint - slot; direction.y = 0; direction.Normalize();
                task.outputOrigin = station.m_spawnPoint != null ? station.m_spawnPoint.position : slot + direction * .5f;
                if (owner != ZNet.GetUID())
                {
                    if (owner == 0) { task.deliveryFailure = "cooking removal has no simulating peer"; return; }
                    task.remoteOutput = new CookingRemoteOutput(task.Output, owner, ZNet.GetUID(), station.m_recordCrafter, count,
                        ItemDrop.s_instances.Where(d => d != null && d.m_nview != null && d.m_nview.IsValid()).Select(d => Perceiver.Id(d.m_nview)));
                }
                task.Trace("normal cooking removal request observed", new Dictionary<string, object>
                { ["owner"] = owner, ["requestedCount"] = count, ["evidence"] = task.remoteOutput == null ? "direct_spawn" : "remote_correlation" });
            }
        }

        [HarmonyPatch(typeof(ItemDrop), "Awake")]
        static class RemoteOutputAppearance
        {
            static void Postfix(ItemDrop __instance) { Active?.ObserveRemoteOutput(__instance); }
        }
    }
}
