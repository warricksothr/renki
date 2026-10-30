using HarmonyLib;
using Renki.Movement;
using UnityEngine;

namespace Renki.Plugin
{
    /// <summary>
    /// The bot's hands on the controls. Body tasks write what they want here;
    /// the PlayerController patches below feed it to Player.SetControls every
    /// physics step instead of the keyboard, so the bot works whether or not
    /// the game window has focus (docs/BRIDGE-PROTOCOL.md 7.3 and section 10).
    /// While Active is false the game reads its own input as usual.
    /// </summary>
    public static class Driver
    {
        /// <summary>The controller holds a session: the bot has the controls.</summary>
        public static bool Active;

        /// <summary>The world-space direction to move in; zero stands still.</summary>
        public static Vector3 Move;

        public static bool Run;

        // Harvest is the only task that holds normal primary attack input.
        // Stand/cancel clears it; no queued click survives task replacement.
        internal static bool HarvestAttack;
        internal static bool MeleeAttack, Block, Dodge;

        /// <summary>Jump on the next physics step; cleared once used.</summary>
        public static bool Jump;

        /// <summary>A point to face instead of the move direction, until LookUntil.</summary>
        public static Vector3? LookAt;

        // Harvest aims its normal attack from the tool's attack height,
        // including pitch for a low stump or resource on a slope.
        internal static float? LookOriginHeight;

        public static float LookUntil;

        /// <summary>Runs first in every controlled physics step: the task runner's tick.</summary>
        public static System.Action<float> OnFixedStep;

        /// <summary>Degrees per second the bot turns at.</summary>
        public const float TurnRate = 540f;

        public static void Stand()
        {
            Move = Vector3.zero;
            Run = false;
            Jump = false;
            HarvestAttack = false;
            MeleeAttack = Block = Dodge = false;
            LookOriginHeight = null;
        }

        internal static void Step(Player player, float dt)
        {
            OnFixedStep?.Invoke(dt);

            Vector3 want = Vector3.zero;
            var pitch = LookAt.HasValue && Time.time < LookUntil && LookOriginHeight.HasValue;
            if (LookAt.HasValue && Time.time < LookUntil)
                want = LookAt.Value - (pitch ? player.transform.position + Vector3.up * LookOriginHeight.Value : player.GetHeadPoint());
            else if (Move.sqrMagnitude > 0.0001f)
                want = Move;
            if (!pitch) want.y = 0;
            if (want.sqrMagnitude > 0.0001f)
            {
                var current = player.m_lookDir;
                if (!pitch) current.y = 0;
                var turned = Vector3.RotateTowards(current.normalized, want.normalized, TurnRate * Mathf.Deg2Rad * dt, 0f);
                player.SetLookDir(turned);
            }

            var local = Controls.ToLocal(Convert.ToV3(Move), Convert.ToV3(player.m_lookDir));
            // The user's toggle-block preference must not latch bot blocking.
            var toggleBlock = player.ToggleBlock;
            player.ToggleBlock = false;
            try
            {
                player.SetControls(new Vector3((float)local.X, 0, (float)local.Z),
                    attack: false, attackHold: HarvestAttack || MeleeAttack, secondaryAttack: false, secondaryAttackHold: false,
                    block: false, blockHold: Block, jump: Jump, crouch: false, run: Run, autoRun: false, dodge: Dodge);
            }
            finally { player.ToggleBlock = toggleBlock; }
            Jump = Dodge = false;
        }

        internal static void ClearCombat()
        {
            Stand(); LookAt = null; LookUntil = 0;
            var p = Player.m_localPlayer;
            if (p == null) return;
            // Normal input queues a dodge for up to half a second. Replacement
            // must not execute it after a new task or an owner's stop.
            p.m_queuedDodgeTimer = p.m_queuedAttackTimer = p.m_queuedSecondAttackTimer = 0;
            p.m_attack = p.m_attackHold = p.m_secondaryAttack = p.m_secondaryAttackHold = p.m_blocking = false;
        }
    }

    [HarmonyPatch(typeof(PlayerController), "FixedUpdate")]
    static class PlayerControllerFixedUpdate
    {
        static bool Prefix(PlayerController __instance)
        {
            if (!Driver.Active) return true;
            var player = __instance.m_character;
            if (player == null || player != Player.m_localPlayer) return true;
            if (__instance.m_nview != null && !__instance.m_nview.IsOwner()) return false;
            Driver.Step(player, Time.fixedDeltaTime);
            return false;
        }
    }

    /// <summary>Mouse look would fight the bot's steering, so it is off while the bot drives.</summary>
    [HarmonyPatch(typeof(PlayerController), "LateUpdate")]
    static class PlayerControllerLateUpdate
    {
        static bool Prefix(PlayerController __instance) =>
            !Driver.Active || __instance.m_character != Player.m_localPlayer;
    }

    static class Convert
    {
        public static V3 ToV3(Vector3 v) => new V3(v.x, v.y, v.z);
        public static Vector3 ToVector3(V3 v) => new Vector3((float)v.X, (float)v.Y, (float)v.Z);
        public static double[] ToArray(Vector3 v) => ToV3(v).ToArray();
    }
}
