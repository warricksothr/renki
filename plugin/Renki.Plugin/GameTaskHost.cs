using System;
using Renki.Protocol;
using Renki.Tasks;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>The tasks, carried out in the game. Everything here runs on
    /// Unity's main thread.</summary>
    sealed class GameTaskHost : ITaskHost
    {
        /// <summary>How long a look_at holds before the bot faces where it walks again.</summary>
        const float LookHold = 4f;

        public IBodyTask StartBody(TaskRequest req, object args)
        {
            if (Player.m_localPlayer == null || Player.m_localPlayer.IsDead())
                throw new TaskRefused(Wire.ReasonStopped, "not in the world");
            switch (args)
            {
                case MoveToArgs m:
                    {
                        var (pos, obj) = Resolve(m.To);
                        return new MoveToTask(m, pos, obj);
                    }
                case PickupArgs pk:
                    {
                        var obj = FindEntity(pk.Entity);
                        if (obj == null || !obj.TryGetComponent<ItemDrop>(out var drop) || drop.m_itemData == null)
                            throw new TaskRefused(Wire.ReasonInvalidTarget, $"no drop {pk.Entity} in range");
                        // Not even one fits: the game would leave it lying there.
                        if (!Player.m_localPlayer.GetInventory().CanAddItem(drop.m_itemData, 1))
                            throw new TaskRefused(Wire.ReasonInventoryFull, $"no room for {drop.m_itemData.m_shared.m_name}");
                        return new PickupTask(drop);
                    }
                case FollowArgs f:
                    if (Players.Find(f.Player) == null)
                        throw new TaskRefused(Wire.ReasonInvalidTarget, $"no player \"{f.Player}\" in range");
                    return new FollowTask(f);
            }
            throw new TaskRefused(Wire.ReasonUnsupported, req.Task);
        }

        public Outcome RunInstant(TaskRequest req, object args)
        {
            var player = Player.m_localPlayer;
            if (player == null) return Outcome.Failed(Wire.ReasonStopped, "not in the world");
            switch (args)
            {
                case LookAtArgs l:
                    {
                        var (pos, _) = Resolve(l.To);
                        Driver.LookAt = pos;
                        Driver.LookUntil = Time.time + LookHold;
                        return Outcome.Succeeded();
                    }
                case SayArgs s:
                    if (Chat.instance == null) return Outcome.Failed(Wire.ReasonStopped, "chat is not up");
                    // Valheim's whisper is a quiet line heard nearby; it has no recipient.
                    var type = s.Channel == Wire.ChannelShout ? Talker.Type.Shout
                        : s.Channel == Wire.ChannelWhisper ? Talker.Type.Whisper : Talker.Type.Normal;
                    Chat.instance.SendText(type, s.Text);
                    return Outcome.Succeeded();
                case EmoteArgs e:
                    if (!Enum.TryParse<Emotes>(e.Name, true, out var emote) || emote == Emotes.Count)
                        return Outcome.Failed(Wire.ReasonBadArgs, $"no emote \"{e.Name}\"");
                    Emote.DoEmote(emote);
                    return Outcome.Succeeded();
            }
            return Outcome.Failed(Wire.ReasonUnsupported, req.Task);
        }

        public void Idle() => Driver.Stand();

        /// <summary>A target as a position, plus the object when it is an entity.</summary>
        static (Vector3, GameObject) Resolve(Target t)
        {
            if (t.Pos.HasValue) return (Convert.ToVector3(t.Pos.Value), null);
            var obj = FindEntity(t.Entity);
            if (obj == null) throw new TaskRefused(Wire.ReasonInvalidTarget, $"no entity {t.Entity} in range");
            return (obj.transform.position, obj);
        }

        static GameObject FindEntity(string id)
        {
            var parts = id.Split(':');
            if (parts.Length == 2 && long.TryParse(parts[0], out var user) && uint.TryParse(parts[1], out var n))
                return ZNetScene.instance?.FindInstance(new ZDOID(user, n));
            // A player may also be named.
            return Players.Find(id)?.gameObject;
        }
    }
}
