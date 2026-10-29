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
            // Every body task walks.
            if (Walker.Stranded(Player.m_localPlayer) is Outcome stranded)
                throw new TaskRefused(stranded.Reason, stranded.Detail);
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
                        // Without a prefab its arrival in the pack cannot be told.
                        if (drop.m_itemData.m_dropPrefab == null)
                            throw new TaskRefused(Wire.ReasonInvalidTarget, $"{pk.Entity} has no prefab to count");
                        // Not even one fits: the game would leave it lying there.
                        if (!Player.m_localPlayer.GetInventory().CanAddItem(drop.m_itemData, 1))
                            throw new TaskRefused(Wire.ReasonInventoryFull, $"no room for {drop.m_itemData.m_shared.m_name}");
                        if (PickupTask.TooHeavy(Player.m_localPlayer, drop.m_itemData))
                            throw new TaskRefused(Wire.ReasonTooHeavy, $"{drop.m_itemData.m_shared.m_name} would take it past its carry weight");
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
                case DropArgs d:
                    return Drop(player, d);
            }
            return Outcome.Failed(Wire.ReasonUnsupported, req.Task);
        }

        public void Idle() => Driver.Stand();

        /// <summary>Drops up to Count of a prefab at the bot's feet, from
        /// stacks it does not have equipped, as a player would. The game
        /// marks a player's own drop so its auto-pickup leaves it there.</summary>
        static Outcome Drop(Player player, DropArgs d)
        {
            var inv = player.GetInventory();
            var stacks = new System.Collections.Generic.List<ItemDrop.ItemData>();
            var worn = false;
            foreach (var item in inv.GetAllItems())
            {
                if (item.m_dropPrefab == null || item.m_dropPrefab.name != d.Prefab) continue;
                if (item.m_equipped) worn = true;
                else stacks.Add(item);
            }
            if (stacks.Count == 0)
                return Outcome.Failed(Wire.ReasonInvalidTarget, worn ? $"only an equipped {d.Prefab}" : $"no {d.Prefab} to drop");
            var left = d.Count;
            // Smallest stacks first, so the fewest are split.
            stacks.Sort((a, b) => a.m_stack.CompareTo(b.m_stack));
            foreach (var item in stacks)
            {
                if (left == 0) break;
                var n = System.Math.Min(left, item.m_stack);
                if (player.DropItem(inv, item, n)) left -= n;
            }
            var dropped = d.Count - left;
            if (dropped == 0) return Outcome.Failed(Wire.ReasonInvalidTarget, $"the game would not drop the {d.Prefab}");
            return Outcome.Succeeded($"dropped {dropped} {d.Prefab}");
        }

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
