using System;
using System.Collections.Generic;
using Renki.Tasks;

namespace Renki.Plugin
{
    /// <summary>Observe the exact selected body's owner callback, never a deletion RPC.</summary>
    sealed class CharacterDeathObservation : IDisposable
    {
        readonly DeathObservation observation;
        readonly string entity;
        bool reported;

        internal CharacterDeathObservation(Character enemy, string entity)
        {
            this.entity = entity;
            var player = Player.m_localPlayer;
            var net = ZNet.instance;
            var scene = ZNetScene.instance;
            var world = net != null ? net.GetWorldName() : null;
            var zdo = enemy.m_nview.GetZDO();
            observation = new DeathObservation(
                callback => enemy.m_onDeath += callback,
                // Managed delegate fields remain accessible after Unity destruction.
                callback => enemy.m_onDeath -= callback,
                () => player != null && ReferenceEquals(player, Player.m_localPlayer) && !player.IsDead() &&
                    net != null && ReferenceEquals(net, ZNet.instance) && net.GetWorldName() == world &&
                    scene != null && ReferenceEquals(scene, ZNetScene.instance),
                () => enemy != null && enemy.m_nview != null && enemy.m_nview.IsValid() &&
                    ReferenceEquals(zdo, enemy.m_nview.GetZDO()) && zdo.m_uid.ToString() == entity &&
                    enemy.m_nview.IsOwner() && enemy.IsDead());
        }

        internal bool Observed
        {
            get
            {
                var observed = observation.Observed;
                // Keep diagnostics out of the game's death callback.
                if (observed && !reported)
                {
                    reported = true;
                    Tracer.Emit("combat", "death_observed", new Dictionary<string, object>
                    { ["entity"] = entity, ["evidence"] = "owner_callback" });
                }
                return observed;
            }
        }
        public void Dispose() => observation.Dispose();
    }
}
