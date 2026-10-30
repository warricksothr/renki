import { humanize } from "./status.js";
export function summarize(ev) {
  const f = ev.frame || {},
    a = ev.attrs || {};
  const reason = f.reason || f.data?.reason || a.reason || f.detail || a.detail;
  let title;
  if (f.type === "task")
    title = `${humanize(typeof f.task === "string" ? f.task : "Task")} · ${humanize(f.status || "requested")}`;
  else if (f.type === "chat")
    title = `${f.speaker || (typeof f.from === "object" ? f.from?.name : f.from) || "Chat"}: ${f.text || f.message || "Message"}`;
  else
    title =
      ev.msg ||
      f.msg ||
      (ev.line || "").replace(/^\S+\s+/, "") ||
      humanize(ev.cat || "Event");
  return {
    title: title.length > 220 ? title.slice(0, 217) + "…" : title,
    reason: reason ? humanize(reason) : "",
  };
}
