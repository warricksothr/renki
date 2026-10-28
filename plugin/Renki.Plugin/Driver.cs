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

        /// <summary>Jump on the next physics step; cleared once used.</summary>
        public static bool Jump;

        /// <summary>A point to face instead of the move direction, until LookUntil.</summary>
        public static Vector3? LookAt;

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
        }

        internal static void Step(Player player, float dt)
        {
            OnFixedStep?.Invoke(dt);

            Vector3 want = Vector3.zero;
            if (LookAt.HasValue && Time.time < LookUntil)
                want = LookAt.Value - player.GetHeadPoint();
            else if (Move.sqrMagnitude > 0.0001f)
                want = Move;
            want.y = 0;
            if (want.sqrMagnitude > 0.0001f)
            {
                var current = player.m_lookDir;
                current.y = 0;
                var turned = Vector3.RotateTowards(current.normalized, want.normalized, TurnRate * Mathf.Deg2Rad * dt, 0f);
                player.SetLookDir(turned);
            }

            var local = Controls.ToLocal(Convert.ToV3(Move), Convert.ToV3(player.m_lookDir));
            player.SetControls(new Vector3((float)local.X, 0, (float)local.Z),
                attack: false, attackHold: false, secondaryAttack: false, secondaryAttackHold: false,
                block: false, blockHold: false, jump: Jump, crouch: false, run: Run, autoRun: false, dodge: false);
            Jump = false;
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
