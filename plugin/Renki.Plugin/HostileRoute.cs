using System.Collections.Generic;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>One loaded-threat sample per physics frame, shared by all
    /// planning probes. Passive animals do not acquire an exclusion area.</summary>
    sealed class HostileRoute
    {
        readonly List<Character> characters = new List<Character>();
        readonly List<V3> threats = new List<V3>();
        int frame = -1;
        bool saturated, available;
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
                        threats.Add(Convert.ToV3(c.transform.position));
                available = true;
            }
            return available && !saturated && HostileClearance.Clear(from, to, threats);
        }
    }
}
