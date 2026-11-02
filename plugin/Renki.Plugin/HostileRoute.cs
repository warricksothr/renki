using System.Collections.Generic;
using System.Linq;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One loaded-threat sample per physics frame, shared by all
    /// planning probes. Passive animals do not acquire an exclusion area.</summary>
    sealed class HostileRoute
    {
        readonly List<Character> characters = new List<Character>();
        readonly List<HostileClearance.Threat> threats = new List<HostileClearance.Threat>();
        int frame = -1;
        bool saturated, available;

        // A route can avoid loaded threats beyond the normal perception
        // radius. Report a bounded explanation when those threats block it.
        internal object Snapshot()
        {
            var player = Player.m_localPlayer;
            return new Dictionary<string, object>
            {
                ["available"] = available,
                ["saturated"] = saturated,
                ["count"] = threats.Count,
                ["nearest"] = player == null ? new object[0] : characters
                    .Where(c => c != null && c != player && !c.IsDead() &&
                        c.GetComponent<MonsterAI>() != null && BaseAI.IsEnemy(player, c))
                    .OrderBy(c => (c.transform.position - player.transform.position).sqrMagnitude)
                    .Take(8).Select(c => (object)new Dictionary<string, object>
                    {
                        ["entity"] = c.m_nview != null && c.m_nview.IsValid() ? c.m_nview.GetZDO().m_uid.ToString() : null,
                        ["prefab"] = c.m_nview != null && c.m_nview.IsValid() ? c.m_nview.GetPrefabName() : null,
                        ["radius"] = HostileClearance.RadiusFor(c.m_nview != null && c.m_nview.IsValid() ? c.m_nview.GetPrefabName() : null, c.GetLevel()),
                        ["pos"] = Convert.ToArray(c.transform.position),
                    }).ToArray(),
            };
        }
        internal bool Clear(V3 from, V3 to)
        {
            if (frame != Time.frameCount)
            {
                frame = Time.frameCount; characters.Clear(); threats.Clear(); available = false;
                var player = Player.m_localPlayer;
                if (player == null) return false;
                Character.GetCharactersInRange(player.transform.position, 64, characters);
                saturated = characters.Count > 128;
                if (saturated) return false;
                foreach (var c in characters)
                    if (c != null && c != player && !c.IsDead() && c.GetComponent<MonsterAI>() != null && BaseAI.IsEnemy(player, c))
                        threats.Add(new HostileClearance.Threat(Convert.ToV3(c.transform.position),
                            c.m_nview != null && c.m_nview.IsValid() ? c.m_nview.GetPrefabName() : null, c.GetLevel()));
                available = true;
            }
            return available && !saturated && HostileClearance.Clear(from, to, threats);
        }
    }
}
