import { summarize } from "./timeline.js";
export function TimelineRecord({ event: ev, openTask }) {
  const { title, reason } = summarize(ev);
  return (
    <>
      <div class="event-row">
        <time dateTime={ev.at}>
          {ev.at
            ? new Date(ev.at).toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
                second: "2-digit",
              })
            : "—"}
        </time>
        <span class="event-type">{ev.cat}</span>
        <div class="event-copy">
          <strong>{title}</strong>
          {reason && <p>{reason}</p>}
        </div>
        {ev.task && (
          <button
            class="task-link"
            onClick={() => openTask(ev.task, ev.at)}
            title={`Everything about ${ev.task}`}
          >
            {ev.task} ↗
          </button>
        )}
      </div>
      <details class="event-details">
        <summary>Technical details</summary>
        <pre>{ev.line || JSON.stringify(ev, null, 2)}</pre>
      </details>
    </>
  );
}
