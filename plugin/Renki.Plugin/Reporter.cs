using System.Collections.Generic;
using System.Linq;
using HarmonyLib;
using Renki.Protocol;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// What the controller hears about: the bot's own state (6.1), chat
    /// (6.4) and events (6.5). Runs on the main thread.
    /// </summary>
    sealed class Reporter
    {
        readonly RenkiPlugin plugin;
        Player watched;
        bool wasDead;
        bool lowHealth;
        string lastAttacker;
        string biome;
        HashSet<string> players;
        float nextRoster;

        public Reporter(RenkiPlugin plugin)
        {
            this.plugin = plugin;
        }

        public void Update(bool inSession)
        {
            var p = Player.m_localPlayer;
            if (p != watched)
            {
                if (watched != null)
                {
                    watched.m_onDamaged -= OnDamaged;
                    watched.m_onDeath -= OnDeath;
                }
                watched = p;
                if (p != null)
                {
                    p.m_onDamaged += OnDamaged;
                    p.m_onDeath += OnDeath;
                    if (wasDead && inSession)
                        plugin.Send(new Protocol.Event { Name = Wire.EventRespawned, Pos = Convert.ToArray(p.transform.position) });
                    wasDead = false;
                    lowHealth = false;
                }
                else
                {
                    players = null;
                    biome = null;
                }
            }
            if (p == null || !inSession) return;

            var frac = p.GetMaxHealth() > 0 ? p.GetHealth() / p.GetMaxHealth() : 1;
            if (!lowHealth && frac < 0.3f && !p.IsDead())
            {
                lowHealth = true;
                plugin.Send(new Protocol.Event { Name = Wire.EventLowHealth, HpFrac = System.Math.Round(frac, 2) });
            }
            else if (frac >= 0.3f) lowHealth = false;

            var b = Heightmap.FindBiome(p.transform.position).ToString();
            if (biome != null && b != biome) plugin.Send(new Protocol.Event { Name = Wire.EventEnteredBiome, Biome = b });
            biome = b;

            if (Time.unscaledTime >= nextRoster)
            {
                nextRoster = Time.unscaledTime + 1f;
                Roster();
            }
        }

        /// <summary>player_joined and player_left, from the server's player list.
        /// The first list after joining is the baseline, not a flood of joins.</summary>
        void Roster()
        {
            if (ZNet.instance == null) return;
            var now = ZNet.instance.GetPlayerList()
                .Where(i => i.m_name != null)
                .ToDictionary(i => i.m_name, i => i.m_characterID.UserID.ToString());
            var self = Player.m_localPlayer?.GetPlayerName();
            now.Remove(self ?? "");
            if (players != null)
            {
                foreach (var kv in now.Where(kv => !players.Contains(kv.Key)))
                    plugin.Send(new Protocol.Event { Name = Wire.EventPlayerJoined, PlayerName = kv.Key, PlayerId = kv.Value });
                foreach (var gone in players.Where(n => !now.ContainsKey(n)))
                    plugin.Send(new Protocol.Event { Name = Wire.EventPlayerLeft, PlayerName = gone, PlayerId = "" });
            }
            players = new HashSet<string>(now.Keys);
        }

        void OnDamaged(float amount, Character attacker)
        {
            lastAttacker = attacker != null ? attacker.GetZDOID().ToString() : null;
            plugin.Send(new Protocol.Event
            {
                Name = Wire.EventDamaged,
                By = lastAttacker ?? "",
                Amount = System.Math.Round(amount, 2),
            });
        }

        void OnDeath()
        {
            wasDead = true;
            plugin.Send(new Protocol.Event { Name = Wire.EventDied, By = lastAttacker ?? "" });
            plugin.BotDied();
        }

        public State State(string phase, TaskRef task)
        {
            var p = Player.m_localPlayer;
            var s = new State { Phase = phase, Task = task };
            if (p == null) return s;
            s.Character = p.GetPlayerName();
            s.World = ZNet.instance != null ? ZNet.instance.GetWorldName() ?? "" : "";
            s.Pos = Convert.ToArray(p.transform.position);
            s.Yaw = System.Math.Round(p.transform.rotation.eulerAngles.y, 1);
            s.Vel = Convert.ToArray(p.GetVelocity());
            s.Hp = Round(p.GetHealth());
            s.HpMax = Round(p.GetMaxHealth());
            s.Stamina = Round(p.GetStamina());
            s.StaminaMax = Round(p.GetMaxStamina());
            s.Eitr = Round(p.GetEitr());
            s.EitrMax = Round(p.GetMaxEitr());
            if (p.IsOnGround()) s.Flags.Add("grounded");
            if (p.IsRunning()) s.Flags.Add("running");
            if (p.IsCrouching()) s.Flags.Add("crouching");
            if (p.IsSwimming()) s.Flags.Add("swimming");
            if (p.IsSitting()) s.Flags.Add("sitting");
            if (p.IsSleeping()) s.Flags.Add("sleeping");
            if (p.InAttack()) s.Flags.Add("attacking");
            if (p.IsBlocking()) s.Flags.Add("blocking");
            if (p.IsEncumbered()) s.Flags.Add("encumbered");
            // The same test the walker fails a body task on.
            if (p.IsEncumbered() && !p.HaveStamina()) s.Flags.Add("stranded");
            if (p.InShelter()) s.Flags.Add("in_shelter");
            s.Biome = Heightmap.FindBiome(p.transform.position).ToString();
            var env = EnvMan.instance;
            if (env != null)
            {
                s.Env = env.GetCurrentEnvironment()?.m_name ?? "";
                s.Time = new GameTime
                {
                    Day = env.GetDay(),
                    DayFraction = System.Math.Round(env.GetDayFraction(), 3),
                    Night = EnvMan.IsNight(),
                };
            }
            Equip(s, "right", p.m_rightItem);
            Equip(s, "left", p.m_leftItem);
            Equip(s, "chest", p.m_chestItem);
            Equip(s, "legs", p.m_legItem);
            Equip(s, "helmet", p.m_helmetItem);
            Equip(s, "shoulder", p.m_shoulderItem);
            Equip(s, "utility", p.m_utilityItem);
            foreach (var se in p.GetSEMan().GetStatusEffects())
            {
                if (se == null) continue;
                s.Effects.Add(se.name.Replace("(Clone)", ""));
                if (se.name.StartsWith("CampFire")) s.Flags.Add("near_fire");
            }
            s.Weight = Round(p.GetInventory().GetTotalWeight());
            s.WeightMax = Round(p.GetMaxCarryWeight());
            return s;
        }

        static double Round(float f) => System.Math.Round(f, 1);

        static void Equip(State s, string slot, ItemDrop.ItemData item)
        {
            var prefab = item?.m_dropPrefab != null ? item.m_dropPrefab.name : null;
            if (prefab != null) s.Equipped[slot] = prefab;
        }
    }

    /// <summary>Every chat line except the bot's own goes to the controller (6.4).</summary>
    [HarmonyPatch(typeof(Chat), nameof(Chat.OnNewChatMessage))]
    static class ChatPatch
    {
        static void Postfix(long senderID, Vector3 pos, Talker.Type type, UserInfo sender, string text)
        {
            var plugin = RenkiPlugin.Instance;
            if (plugin == null || ZNet.instance == null || sender == null) return;
            if (senderID == ZNet.instance.LocalPlayerCharacterID.UserID) return;
            var me = Player.m_localPlayer;
            var name = sender.Name ?? "";
            if (type != Talker.Type.Ping) pos = Feet(name, pos);
            plugin.Send(new Protocol.Chat
            {
                From = new ChatSender { Name = name, PlayerId = senderID.ToString(), Owner = plugin.Owners.Contains(name) },
                Channel = type == Talker.Type.Shout ? Wire.ChannelShout
                    : type == Talker.Type.Whisper ? Wire.ChannelWhisper
                    : type == Talker.Type.Ping ? Wire.ChannelPing : Wire.ChannelNormal,
                Text = text ?? "",
                Pos = Convert.ToArray(pos),
                Dist = me != null ? System.Math.Round(Vector3.Distance(me.transform.position, pos), 1) : 0,
            });
        }

        static readonly int Ground = LayerMask.GetMask("Default", "static_solid", "terrain", "piece", "vehicle");

        /// <summary>The game sends a speaker's head; where they stand is what
        /// "come here" and "set home" mean. Their feet, when their character
        /// is loaded near the head, else the ground under the head.</summary>
        static Vector3 Feet(string name, Vector3 head)
        {
            foreach (var p in Player.GetAllPlayers())
                if (p.GetPlayerName() == name && Vector3.Distance(p.GetHeadPoint(), head) < 3)
                    return p.transform.position;
            return Physics.Raycast(head, Vector3.down, out var hit, 3f, Ground) ? hit.point : head;
        }
    }
}
