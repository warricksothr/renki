using Renki.Protocol;

namespace Renki.Plugin
{
    /// <summary>Observe game-owned events; never infer a raid from enemy count.</summary>
    static class WorldEvents
    {
        public static WorldEventStatus Capture(double[] position)
        {
            var system = RandEventSystem.instance;
            if (system == null) return null;
            var status = new WorldEventStatus();
            var random = system.GetCurrentRandomEvent();
            if (random != null)
            {
                var area = new AreaEvent
                {
                    Name = random.m_name,
                    Message = Message(random),
                    Center = Convert.ToArray(random.m_pos),
                    Radius = random.m_eventRange,
                    ElapsedS = random.GetTime(),
                    DurationS = random.m_duration,
                    TimerMayPause = random.m_pauseIfNoPlayerInArea,
                };
                if (!area.Valid()) return null;
                area.Inside = area.Contains(position);
                status.RandomEvent = area;
            }
            var active = system.GetActiveEvent();
            if (active != null)
                status.LocalEvent = new EventNotice
                {
                    Name = active.m_name,
                    Message = Message(active),
                    Kind = ReferenceEquals(active, random) ? "random" : "local",
                };
            return status;
        }

        static string Message(RandomEvent e)
        {
            var text = e.GetHudText() ?? "";
            return Localization.instance != null ? Localization.instance.Localize(text) : text;
        }
    }
}
