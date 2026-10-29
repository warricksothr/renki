namespace Renki.Movement
{
    /// <summary>What a climb wants this step.</summary>
    public enum ClimbStep { Wait, Jump, Push, Done }

    /// <summary>
    /// A climb out of a stuck spot: stand still until stamina has come back
    /// far enough for several jumps in a row, then jump that many times a
    /// moment apart while pushing on, as a player scrambles up a ledge or
    /// over a pile. A single hop on an empty stamina bar does neither. It
    /// waits no longer than WaitFor, and not at all when stamina cannot come
    /// back (the game stops it while the bot is overloaded).
    /// </summary>
    public sealed class Climber
    {
        /// <summary>How many jumps in a row.</summary>
        public int Jumps = 3;

        /// <summary>Seconds between jumps: about the time to land.</summary>
        public double Gap = 0.8;

        /// <summary>The longest wait for stamina, in seconds.</summary>
        public double WaitFor = 10;

        double started = double.NaN;
        double nextJump;
        int jumped;
        bool waiting;

        /// <summary>The stamina the jumps need, a little over, and never
        /// more than nearly a full bar.</summary>
        public double Needed(double jumpCost, double staminaMax) =>
            System.Math.Min(Jumps * jumpCost + 1, staminaMax * 0.95);

        public void Start(double now)
        {
            started = now;
            waiting = true;
            jumped = 0;
        }

        public bool Active => !double.IsNaN(started);

        /// <summary>The next step, from the time, the stamina, what a jump
        /// costs, and whether stamina can come back at all.</summary>
        public ClimbStep Tick(double now, double stamina, double staminaMax, double jumpCost, bool canRegen)
        {
            if (!Active) return ClimbStep.Done;
            if (waiting)
            {
                var enough = stamina >= Needed(jumpCost, staminaMax);
                if (!enough && canRegen && now - started < WaitFor) return ClimbStep.Wait;
                waiting = false;
                nextJump = now;
            }
            if (now < nextJump) return ClimbStep.Push;
            // Out of stamina, or done: one last push carries the last jump.
            if (jumped >= Jumps || stamina < jumpCost)
            {
                started = double.NaN;
                return ClimbStep.Done;
            }
            jumped++;
            nextJump = now + Gap;
            return ClimbStep.Jump;
        }

        /// <summary>Stops the climb, as when the bot gets free.</summary>
        public void Stop() => started = double.NaN;
    }
}
