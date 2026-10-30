import { useEffect, useState } from "preact/hooks";
let current = null;
const listeners = new Set();
export function publishStatus(value) {
  current = value;
  for (const listener of listeners) listener(value);
}
export function useStatus() {
  const [status, setStatus] = useState(current);
  useEffect(() => {
    listeners.add(setStatus);
    setStatus(current);
    return () => listeners.delete(setStatus);
  }, []);
  return status;
}
export const online = (s) => !!(s?.connected && s.phase === "in_world");
export const scope = (s) =>
  online(s) ? `${s.connection}:${s.world}:${s.worldUid || ""}` : "";
export const humanize = (text) =>
  String(text || "")
    .replace(/^\$item_/, "")
    .replace(/_/g, " ")
    .replace(/([a-z])([A-Z])/g, "$1 $2");
export const getStatus = () => current;
export function subscribeStatus(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
