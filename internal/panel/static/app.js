// The renki panel: one event stream (/api/events) feeds the header, the
// feed and the map. Records are audit log records (docs/AUDIT.md) with the
// timeline's line and a category added; statuses are /api/status snapshots.
"use strict";

const $ = (id) => document.getElementById(id);
const FEED_MAX = 1500;
const TRAIL_MAX = 600;
const STUCK_MAX = 30;

const world = {
  bot: null, // [x, y, z]
  yaw: 0,
  trail: [],
  characters: [], // from perception
  structures: [],
  resources: [],
  walls: [], // from layout: {a, b, h}
  home: null,
  left: [], // what the bot left on the ground: {pos, items}
  path: null, // {from, to}
  stuck: [], // {pos, attempt}
  tiles: new Map(),
  landmarks: [],
  danger: [],
  waypoints: [],
  containers: new Map(),
};

const layers = { terrain: true, elevation: true, memory: true, danger: true, resources: false };
for (const b of document.querySelectorAll("[data-layer]")) b.addEventListener("click", () => {
  const key = b.dataset.layer;
  layers[key] = !layers[key];
  b.setAttribute("aria-pressed", String(layers[key]));
  draw();
});
let selectedAtlas = "", savedWorlds = [], atlasListFetchedAt = 0;
let atlasPinned = false, mapOnline = false, lastLiveAtlas = "", lastLiveName = "";
let mapWorld = "", mapUID = "", mapConnection = 0, mapGeneration = 0, mapLoading = false, mapFetchedAt = 0;

function resetMap() {
  mapGeneration++;
  Object.assign(world, { bot: null, trail: [], characters: [], structures: [], resources: [], walls: [],
    home: null, left: [], path: null, stuck: [], tiles: new Map(), landmarks: [], danger: [], waypoints: [], containers: new Map() });
  resetPlaceEditor();
  mapFetchedAt = 0;
  $("terrain-status").textContent = "Waiting for terrain";
}

// Auto mode follows Tas live, then keeps his last saved atlas through logout.
// A deliberate saved-world selection stays pinned when he joins elsewhere.
function selectAtlas(key, keepTerrain = false) {
  if (key === selectedAtlas) return;
  const tiles = world.tiles;
  if (keepTerrain && view.followBot && world.bot) view.centre = [world.bot[0],world.bot[2]];
  selectedAtlas = key;
  resetMap(); mapWorld = ""; mapUID = ""; mapConnection = 0;
  if (keepTerrain) world.tiles = tiles;
  const saved = savedWorlds.find(w => w.key === key);
  if (key && saved && !keepTerrain) { view.centre = saved.center; view.zoomed = true; }
  $("atlas-world").value = atlasPinned ? selectedAtlas : "";
  if (key) {
    $("terrain-status").textContent = mapOnline ? "Saved terrain · fogged" : "Bot offline · remembered terrain · fogged";
    $("map-name").textContent = "Remembered terrain";
    $("map-world").textContent = `${saved?.name || lastLiveName || "Saved world"} · ${mapOnline ? "saved atlas" : "bot offline"}`;
    for (const id of ["biome","surface","shelter","pos","height","depth"]) $(id).textContent = "";
  }
  draw();
}
function followAtlas() {
  if (atlasPinned) return;
  const key = mapOnline ? "" : lastLiveAtlas || savedWorlds[0]?.key || "";
  selectAtlas(key, !mapOnline && key === lastLiveAtlas && world.tiles.size > 0);
}

async function loadAtlasWorlds() {
  atlasListFetchedAt = Date.now();
  try {
    const res = await fetch("/api/atlas/worlds", { cache: "no-store" });
    if (!res.ok) return;
    const data = await res.json();
    if (!Array.isArray(data)) return;
    savedWorlds = data;
    const select = $("atlas-world");
    select.replaceChildren(Object.assign(document.createElement("option"), { value: "", textContent: "Follow Tas · last map offline" }));
    for (const w of data) select.append(Object.assign(document.createElement("option"), {
      value: w.key, textContent: `${w.name} · ${w.scope.namespace} · epoch ${w.scope.epoch} · ${w.scope.worldUid}` }));
    if (selectedAtlas && !data.some(w => w.key === selectedAtlas)) select.append(Object.assign(document.createElement("option"), {value:selectedAtlas,textContent:"Atlas no longer retained"}));
    followAtlas();
    select.value = atlasPinned ? selectedAtlas : "";
    if (selectedAtlas) loadMap();
  } catch { /* A transient list failure leaves the current selection alone. */ }
}

$("atlas-world").addEventListener("change", () => {
  const key = $("atlas-world").value;
  atlasPinned = Boolean(key);
  if (atlasPinned) {
    selectAtlas(key);
    view.followBot = false; $("follow-bot").checked = false;
  } else {
    view.followBot = true; $("follow-bot").checked = true;
    followAtlas();
  }
  const saved = savedWorlds.find(w => w.key === selectedAtlas);
  $("map-name").textContent = selectedAtlas ? "Remembered terrain" : "Location";
  $("map-world").textContent = saved ? `${saved.name} · ${mapOnline ? "saved atlas" : "bot offline"}` : "";
  loadMap(); draw();
});

async function loadMap() {
  if ((!mapWorld && !selectedAtlas) || mapLoading) return;
  mapLoading = true;
  const generation = mapGeneration, selected = selectedAtlas;
  const centre = !selected && view.followBot && world.bot ? [world.bot[0], world.bot[2]] : view.centre;
  const rect = $("map").getBoundingClientRect();
  const halfX = Math.min(256, rect.width / (2 * view.scale) + 16), halfZ = Math.min(256, rect.height / (2 * view.scale) + 16);
  const params = new URLSearchParams({ world: selected || "live", minX: centre[0]-halfX, maxX: centre[0]+halfX,
    minZ: centre[1]-halfZ, maxZ: centre[1]+halfZ });
  try {
    const [liveResponse, atlasResponse] = await Promise.all([
      selected ? null : fetch("/api/map", { cache: "no-store" }),
      fetch("/api/atlas/map?" + params, { cache: "no-store" }),
    ]);
    const data = liveResponse?.ok ? await liveResponse.json() : null;
    const atlas = atlasResponse.ok ? await atlasResponse.json() : null;
    if (generation !== mapGeneration || selected !== selectedAtlas) return;
    if (selected) {
      if (atlasResponse.status === 404) { world.tiles.clear(); $("terrain-status").textContent="Atlas no longer retained"; draw(); return; }
      if (!atlas || atlas.worldKey !== selected) return;
      world.tiles = new Map((atlas.tiles || []).map((t) => [RenkiTerrain.key(t.x,t.z), t]));
    } else {
      if (!data || data.world !== mapWorld || data.connection !== mapConnection || (data.worldUid || "") !== mapUID) return;
      const saved = atlas && atlas.connection === mapConnection && atlas.worldKey === data.worldKey ? atlas.tiles || [] : [];
      lastLiveAtlas = data.worldKey || ""; lastLiveName = data.world;
      world.tiles = RenkiTerrain.merge(data.tiles || [], saved);
      world.landmarks = data.landmarks || [];
      world.danger = data.danger || [];
      world.waypoints = data.waypoints || [];
      syncPlaceList();
      world.containers = new Map((data.containers || []).map((c) => [c.entity, c]));
      if (data.perception) {
        world.structures = data.perception.structures || [];
        world.resources = data.perception.resources || [];
        world.characters = data.perception.characters || [];
      }
      world.walls = data.walls || [];
    }
    mapFetchedAt = Date.now();
    $("terrain-status").textContent = world.tiles.size
      ? `${selected ? mapOnline ? "Saved terrain · fogged" : "Bot offline · remembered terrain · fogged" : "Terrain"} · 4 m samples · faded = stale${halfX === 256 || halfZ === 256 ? " · detail within 512 m" : ""}`
      : selected ? "No retained samples here" : "Waiting for terrain";
    draw();
  } catch { /* Keep the last scoped snapshot through a transient request failure. */ }
  finally { mapLoading = false; }
}

const view = {
  scale: 4, // pixels per metre
  centre: [0, 0], // x, z
  followBot: true,
  // Until the user zooms, the scale fits the perception radius.
  zoomed: false,
  radius: 0,
};

// ---- feed ----

const feed = $("feed");
const shown = new Set(
  [...document.querySelectorAll(".filters input[data-cat]")].filter((c) => c.checked).map((c) => c.dataset.cat),
);
let lastStateLine = 0;

for (const box of document.querySelectorAll(".filters input[data-cat]")) {
  box.addEventListener("change", () => {
    box.checked ? shown.add(box.dataset.cat) : shown.delete(box.dataset.cat);
    for (const li of [...feed.children, ...$("history").children])
      li.classList.toggle("hidden", !!li.dataset.cat && !shown.has(li.dataset.cat));
  });
}

function addLine(ev) {
  if (ev.cat === "perception") return;
  if (ev.cat === "state") {
    const now = Date.parse(ev.at);
    if (now - lastStateLine < 1000) return;
    lastStateLine = now;
  }
  const li = lineItem(ev);
  if (!shown.has(ev.cat)) li.classList.add("hidden");
  const atBottom = feed.scrollTop + feed.clientHeight >= feed.scrollHeight - 20;
  feed.append(li);
  while (feed.children.length > FEED_MAX) feed.firstChild.remove();
  if (atBottom && !$("pause").checked) feed.scrollTop = feed.scrollHeight;
}

// lineItem is one record as a line, with a link to its task's view.
function lineItem(ev) {
  const li = document.createElement("li");
  li.dataset.cat = ev.cat;
  li.className = ev.cat;
  if (ev.level === "WARN") li.classList.add("warn");
  if (ev.level === "ERROR") li.classList.add("error");
  if (ev.task) {
    const b = document.createElement("button");
    b.className = "task-link";
    b.textContent = ev.task;
    b.title = `Everything about ${ev.task}`;
    b.addEventListener("click", () => openTask(ev.task, ev.at));
    li.append(b);
  }
  li.append(ev.line);
  return li;
}

// ---- saying things to the bot ----

function sayAs(owners) {
  const sel = $("say-as");
  if (sel.dataset.owners === owners.join("\n")) return;
  sel.dataset.owners = owners.join("\n");
  const keep = sel.value;
  sel.replaceChildren(...owners.map((o) => new Option(o, o)));
  if (owners.includes(keep)) sel.value = keep;
}

$("say-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const text = $("say-text").value.trim();
  if (!text) return;
  const res = await fetch("/api/say", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Renki-Act": "1" },
    body: JSON.stringify({ as: $("say-as").value, text }),
  }).catch(() => null);
  const body = res ? await res.json().catch(() => ({})) : {};
  // The bot's answer comes back in the feed.
  $("say-result").textContent = res?.ok ? "" : body.text || "the controller did not answer";
  if (res?.ok) $("say-text").value = "";
});

// ---- past days and tasks ----

async function loadDays() {
  const res = await fetch("/api/days").catch(() => null);
  if (!res || !res.ok) return;
  const sel = $("day");
  const keep = sel.value;
  const days = await res.json();
  sel.replaceChildren(new Option("Live", ""), ...days.map((d) => new Option(`${d.day} (${(d.bytes / 1e6).toFixed(0)} MB)`, d.day)));
  sel.value = keep;
}

// showDay reads a day's records into the history list, newest last; before
// pages back to earlier ones.
let dayRead = 0; // which read is the newest, so a late answer is dropped

async function showDay(day, before, skip) {
  const hist = $("history");
  const read = ++dayRead; // going back to live ends a read in flight too
  const live = day === "";
  feed.hidden = !live;
  hist.hidden = live;
  if (live) return true;
  if (!before) hist.replaceChildren(Object.assign(document.createElement("li"), { textContent: "Reading…" }));
  const q = new URLSearchParams({ day });
  if (before) q.set("before", before);
  if (skip) q.set("skip", skip);
  const res = await fetch(`/api/timeline?${q}`).catch(() => null);
  if (read !== dayRead) return false;
  if (!res || !res.ok) {
    const err = Object.assign(document.createElement("li"), {
      textContent: res ? await res.text() : "the controller did not answer",
      className: "error",
    });
    // A failed page back keeps what is shown, and its button to try again.
    before ? hist.prepend(err) : hist.replaceChildren(err);
    return false;
  }
  const tl = await res.json();
  if (read !== dayRead) return false;
  const items = tl.events.filter((ev) => ev.cat !== "perception").map((ev) => {
    const li = lineItem(ev);
    if (!shown.has(ev.cat)) li.classList.add("hidden");
    return li;
  });
  if (tl.more && tl.events.length) {
    const more = document.createElement("li");
    const b = document.createElement("button");
    b.className = "earlier";
    b.textContent = "Load earlier";
    const oldest = tl.events[0].at;
    // How many records of the oldest instant are shown, with any shown
    // from it before.
    const atOldest = tl.events.filter((ev) => ev.at === oldest).length + (before === oldest ? skip || 0 : 0);
    b.addEventListener("click", async () => {
      b.disabled = true;
      if (await showDay(day, oldest, atOldest)) more.remove();
      else b.disabled = false;
    });
    more.append(b);
    items.unshift(more);
  }
  if (before) {
    hist.prepend(...items);
  } else {
    hist.replaceChildren(...items);
    hist.scrollTop = hist.scrollHeight;
  }
  return true;
}

let taskRead = 0;

async function openTask(id, at) {
  const dlg = $("task-view");
  const read = ++taskRead;
  $("task-title").textContent = `${id}: reading…`;
  $("task-lines").replaceChildren();
  if (!dlg.open) dlg.showModal();
  const res = await fetch(`/api/task?${new URLSearchParams({ id, at })}`).catch(() => null);
  if (read !== taskRead) return;
  if (!res || !res.ok) {
    $("task-title").textContent = `${id}: ${res ? await res.text() : "the controller did not answer"}`;
    return;
  }
  const evs = await res.json();
  if (read !== taskRead) return;
  $("task-lines").replaceChildren(...evs.map(lineItem));
  $("task-title").textContent = taskTitle(id, evs);
}

// taskTitle says what the task was, how it ended and how long it took.
function taskTitle(id, evs) {
  let name = "", end = "", first, last;
  for (const ev of evs) {
    const f = ev.frame;
    if (ev.task !== id) continue;
    first ??= Date.parse(ev.at);
    last = Date.parse(ev.at);
    if (f?.type === "task" && f.id === id) name = f.task;
    if (f?.type === "task" && f.re === id && ["succeeded", "failed", "cancelled", "rejected"].includes(f.status))
      end = f.status + (f.reason ? ` (${f.reason})` : "");
  }
  const secs = first != null ? ((last - first) / 1000).toFixed(1) : "?";
  return `${id}${name ? " " + name : ""}: ${end || "no end seen"}, ${secs} s`;
}

$("task-close").addEventListener("click", () => $("task-view").close());
$("day").addEventListener("focus", loadDays);
$("day").addEventListener("change", () => showDay($("day").value));

// ---- records into the map ----

function onRecord(ev) {
  addLine(ev);
  // A change from chat or another page shows here too.
  if (ev.kind === "log" && (ev.msg === "setting changed" || ev.msg === "setting pinned") && $("settings").open)
    loadSettings();
  const f = ev.frame;
  if (!f) return;
  if (selectedAtlas && f.type !== "inventory") return;
  switch (f.type) {
    case "state":
      if (f.phase !== "in_world") return;
      if (mapWorld && (f.world !== mapWorld || (f.worldUid || "") !== mapUID)) return;
      world.bot = f.pos;
      world.yaw = f.yaw || 0;
      const last = world.trail[world.trail.length - 1];
      if (!last || Math.hypot(last[0] - f.pos[0], last[2] - f.pos[2]) > 0.3) {
        world.trail.push(f.pos);
        if (world.trail.length > TRAIL_MAX) world.trail.shift();
      }
      $("pos").textContent = `x ${f.pos[0].toFixed(1)}  z ${f.pos[2].toFixed(1)}`;
      // Over weightMax the bot only walks; with no stamina left it cannot move (6.1).
      const flags = f.flags || [];
      if (flags.includes("stranded")) chip("load", "overloaded: can't move", "off");
      else if (flags.includes("encumbered")) chip("load", "overloaded", "off");
      else chip("load", "");
      if (f.weightMax && (f.weight !== weight[0] || f.weightMax !== weight[1])) {
        weight = [f.weight, f.weightMax];
        drawInventory();
      }
      break;
    case "inventory":
      inventory = f;
      drawInventory();
      break;
    case "layout":
      if (!mapWorld) break;
      world.walls = f.walls || [];
      break;
    case "perception":
      if (!mapWorld) break;
      world.characters = f.characters || [];
      world.structures = f.structures || [];
      world.resources = f.resources || [];
      break;
    case "trace":
      if (f.source !== "walker" || !f.data) break;
      if (f.data.from && f.data.to) world.path = { from: f.data.from, to: f.data.to, straight: f.msg.startsWith("no navmesh") };
      if (f.msg.startsWith("stuck") && f.data.center) {
        world.stuck.push({ pos: f.data.center, attempt: f.data.attempt });
        if (world.stuck.length > STUCK_MAX) world.stuck.shift();
      }
      break;
  }
  draw();
}

// ---- inventory ----

let inventory = null; // the newest inventory frame (6.3)
let weight = [0, 0]; // from state

// A prefab as a person would say it: AxeBronze to "Axe Bronze".
const plain = (prefab) => prefab.replace(/^\$item_/, "").replace(/_/g, " ").replace(/([a-z])([A-Z])/g, "$1 $2");

function drawInventory() {
  if (!inventory) return;
  const [w, h] = inventory.size;
  const items = inventory.items || [];
  const free = w * h - items.length;
  let summary = `${free} of ${w * h} slots free`;
  if (weight[1]) summary += ` · weight ${Math.round(weight[0])}/${Math.round(weight[1])}`;
  $("inv-summary").textContent = summary;

  const grid = $("inv-grid");
  grid.style.gridTemplateColumns = `repeat(${w}, minmax(0, 1fr))`;
  const at = new Map(items.map((it) => [`${it.slot[0]},${it.slot[1]}`, it]));
  grid.replaceChildren();
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const cell = document.createElement("div");
      cell.className = "slot";
      const it = at.get(`${x},${y}`);
      if (it) {
        const name = document.createElement("span");
        name.className = "name";
        name.textContent = plain(it.prefab);
        cell.append(name);
        if (it.stack > 1 || it.stackMax) {
          const stack = document.createElement("span");
          stack.className = "stack";
          stack.textContent = it.stackMax ? `${it.stack}/${it.stackMax}` : `${it.stack}`;
          cell.append(stack);
        }
        if (it.durabilityMax) {
          const wear = document.createElement("span");
          const frac = Math.max(0, Math.min(1, it.durability / it.durabilityMax));
          wear.className = "wear" + (frac < 0.25 ? " low" : "");
          wear.style.width = `${frac * 100}%`;
          cell.append(wear);
        }
        if (it.equipped) cell.classList.add("equipped");
        let title = `${plain(it.prefab)} ×${it.stack}`;
        if (it.quality) title += `, quality ${it.quality}`;
        if (it.durabilityMax) title += `, durability ${Math.round(it.durability)}/${Math.round(it.durabilityMax)}`;
        if (it.equipped) title += ", equipped";
        cell.title = title;
      }
      grid.append(cell);
    }
  }
}

// ---- status into the header ----

function chip(id, text, cls) {
  const el = $(id);
  el.textContent = text || "";
  el.className = "chip" + (cls ? " " + cls : "");
}

function onStatus(s) {
  renderGoals(s);
 renderPresence(s);
  const needs = s.connected && s.phase === "in_world" ? s.storageNeeded || [] : [];
  const storage = $("storage-needed");
  storage.hidden = needs.length === 0;
  storage.textContent = needs.map(n => `Storage needed: ${Object.entries(n.items).map(([name, count]) => `${count} ${name}`).join(", ")}. ${n.reason}`).join(" · ");
  mapOnline = Boolean(s.connected && s.phase === "in_world" && s.world);
  followAtlas();
  const active = !selectedAtlas && mapOnline;
  const nextUID = active ? s.worldUid || "" : "";
  const nextWorld = active ? s.world : "", nextConnection = active ? s.connection || 0 : 0;
  if (nextWorld !== mapWorld || nextUID !== mapUID || nextConnection !== mapConnection) {
    resetMap(); mapWorld = nextWorld; mapUID = nextUID; mapConnection = nextConnection;
  }
  if (active) {
    world.bot = s.pos || world.bot; world.yaw = s.yaw || 0;
  }
  if (Date.now() - atlasListFetchedAt > 30000) loadAtlasWorlds();
  if (Date.now() - mapFetchedAt > 5000) loadMap();
  $("map-name").textContent = selectedAtlas ? "Remembered terrain" : active ? s.character || "Location" : "Location";
  $("map-world").textContent = selectedAtlas ? `${savedWorlds.find(w => w.key === selectedAtlas)?.name || lastLiveName || "Saved world"} · ${mapOnline ? "saved atlas" : "bot offline"}` : active ? s.world : "Not in the world";
  chip("biome", active ? plain(s.biome || "Biome unknown") : "");
  const loc = active ? s.location : null, flags = active ? s.flags || [] : [];
  const support = { terrain: "On terrain", structure: "On a structure", ship: "On a ship", other: "Grounded",
    swimming: "Swimming", airborne: "Airborne", liquid: "In liquid", unknown: "Support unknown" };
  chip("surface", active ? (loc ? loc.inWater && loc.support !== "swimming" ? "In shallow water" : support[loc.support] || "Support unknown"
    : flags.includes("swimming") ? "Swimming" : flags.includes("grounded") ? "Grounded" : "Support unknown") : "");
  chip("shelter", flags.includes("in_shelter") ? "Sheltered" : "");
  $("height").textContent = active && s.pos ? `Character ${s.pos[1].toFixed(1)} m · terrain ${loc?.groundHeight != null ? loc.groundHeight.toFixed(1) + " m" : "unknown"}` : "Height unknown";
  $("depth").textContent = loc ? loc.inWater ? `Water above feet ${loc.waterDepth.toFixed(1)} m` : "Feet out of water" : "";
  $("pos").textContent = active && s.pos ? `x ${s.pos[0].toFixed(1)}  z ${s.pos[2].toFixed(1)}` : "";
  chip("link", s.connected ? "connected" : "no plugin", s.connected ? "on" : "off");
  for (const b of document.querySelectorAll("button.act")) b.disabled = !s.connected || acting;
  onModel(s);
  sayAs(s.owners || []);
  chip("who", s.character ? `${s.character}${s.world ? " in " + s.world : ""}` : "");
  chip("phase", s.phase ? s.phase.replace("_", " ") : "");
  chip("task", s.task ? `${s.task.task} (${s.task.id})` : s.connected ? "idle" : "");
  if (s.perceptionRadius) view.radius = s.perceptionRadius;
  chip("hp", s.hpMax ? `hp ${Math.round(s.hp)}/${Math.round(s.hpMax)}` : "");
  chip("stamina", s.staminaMax ? `stamina ${Math.round(s.stamina || 0)}/${Math.round(s.staminaMax)}` : "");
  chip("combat", s.combat ? `combat: ${s.combat.mode}${s.combat.reason ? " · " + s.combat.reason : ""}` : "");
  world.left = active ? s.leftBehind || [] : [];
  if (active && s.home) {
    world.home = s.home.pos;
    chip("home", s.home.distance != null ? `home ${Math.round(s.home.distance)} m` : "home set");
  } else {
    world.home = null;
    chip("home", s.connected ? "no home" : "");
  }
  draw();
}

// ---- map ----

const canvas = $("map");
new ResizeObserver(() => draw()).observe(canvas);
const ctx = canvas.getContext("2d");
const css = getComputedStyle(document.documentElement);
const colour = (name) => css.getPropertyValue("--" + name).trim();
let pending = false;

function draw() {
  if (pending) return;
  pending = true;
  requestAnimationFrame(render);
}

function render() {
  pending = false;
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  if (canvas.width !== w * dpr || canvas.height !== h * dpr) {
    canvas.width = w * dpr;
    canvas.height = h * dpr;
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, w, h);
  // A hidden or collapsed map has no size to fit; keep the scale in range.
  if (!view.zoomed && view.radius > 0 && w > 0 && h > 0)
    view.scale = Math.min(40, Math.max(0.25, Math.min(w, h) / (2 * view.radius * 1.1)));
  if (view.followBot && world.bot) view.centre = [world.bot[0], world.bot[2]];
  // North (+z) is up, east (+x) is right.
  const sx = (x) => w / 2 + (x - view.centre[0]) * view.scale;
  const sy = (z) => h / 2 - (z - view.centre[1]) * view.scale;

  RenkiTerrain.draw(ctx, world.tiles, sx, sy, layers, w, h, Boolean(selectedAtlas));
  grid(w, h, sx, sy);
  if (layers.danger) for (const zone of world.danger) {
    ctx.save();
    ctx.fillStyle = `rgba(230, 82, 68, ${Math.min(0.3, 0.04 + zone.score * 0.035)})`;
    ctx.strokeStyle = "#ed736a99"; ctx.setLineDash([5, 4]);
    ctx.beginPath(); ctx.arc(sx(zone.pos[0]), sy(zone.pos[2]), zone.radius * view.scale, 0, 2 * Math.PI);
    ctx.fill(); ctx.stroke(); ctx.restore();
  }

  for (const p of world.waypoints) {
    if (!(p.radius > 0)) continue;
    ctx.save(); ctx.strokeStyle = "#d4a6ff90"; ctx.fillStyle = "#b08cff0b";
    ctx.setLineDash(p.source === "bot" ? [3, 5] : [8, 4]);
    ctx.beginPath(); ctx.arc(sx(p.pos[0]), sy(p.pos[2]), p.radius * view.scale, 0, 2 * Math.PI);
    ctx.fill(); ctx.stroke(); ctx.restore();
  }

  if (world.bot && view.radius) {
    ctx.save(); ctx.strokeStyle = "#96bdd850"; ctx.setLineDash([3, 6]); ctx.lineWidth = 1;
    ctx.beginPath(); ctx.arc(sx(world.bot[0]), sy(world.bot[2]), view.radius * view.scale, 0, 2 * Math.PI); ctx.stroke(); ctx.restore();
  }

  // Walls under everything else, as thick as they are, near enough. Those
  // standing on a storey above the bot are faint, so the doorways on its
  // own show through.
  if (world.walls.length) {
    ctx.strokeStyle = colour("wall");
    ctx.lineWidth = Math.max(2, 0.3 * view.scale);
    ctx.lineCap = "round";
    const floor = world.bot ? world.bot[1] : null;
    const level = (wl) => floor === null || wl.a[1] - floor < 1.5;
    for (const own of [false, true]) {
      ctx.globalAlpha = own ? 1 : 0.3;
      ctx.beginPath();
      for (const wl of world.walls) {
        if (level(wl) !== own) continue;
        ctx.moveTo(sx(wl.a[0]), sy(wl.a[2]));
        ctx.lineTo(sx(wl.b[0]), sy(wl.b[2]));
      }
      ctx.stroke();
    }
    ctx.globalAlpha = 1;
    ctx.lineCap = "butt";
  }

  if (world.path) {
    ctx.strokeStyle = colour("path");
    ctx.setLineDash(world.path.straight ? [6, 4] : []);
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(sx(world.path.from[0]), sy(world.path.from[2]));
    ctx.lineTo(sx(world.path.to[0]), sy(world.path.to[2]));
    ctx.stroke();
    ctx.setLineDash([]);
  }

  if (world.trail.length > 1) {
    ctx.strokeStyle = colour("bot");
    ctx.globalAlpha = 0.5;
    ctx.lineWidth = 2;
    ctx.beginPath();
    world.trail.forEach((p, i) => (i ? ctx.lineTo(sx(p[0]), sy(p[2])) : ctx.moveTo(sx(p[0]), sy(p[2]))));
    ctx.stroke();
    ctx.globalAlpha = 1;
  }

  ctx.strokeStyle = colour("stuck");
  ctx.lineWidth = 1.5;
  for (const s of world.stuck) {
    ctx.beginPath();
    ctx.arc(sx(s.pos[0]), sy(s.pos[2]), Math.max(4, 3 * view.scale), 0, 2 * Math.PI);
    ctx.stroke();
  }

  // Everything with a place gets a marker; markers of one kind close
  // together on screen become one, with a count. Labels go on in order of
  // importance and are skipped where they would overlap.
  const markers = collect(sx, sy);
  const taken = [];
  if (world.bot) {
    taken.push(box(sx(world.bot[0]) - 12, sy(world.bot[2]) - 12, 24, 24));
    taken.push(box(sx(world.bot[0]) + 13, sy(world.bot[2]) - 10, 26, 16));
  }
  for (const m of markers) {
    ctx.globalAlpha = m.remembered ? .35 : 1;
    drawMarker(m);
    taken.push(box(m.x - m.r, m.y - m.r, 2 * m.r, 2 * m.r));
  }
  ctx.globalAlpha = 1;
  for (const m of [...markers].sort((a, b) => a.prio - b.prio)) {
    ctx.globalAlpha = m.remembered ? .5 : 1;
    if (m.label) place(m, taken);
  }
  ctx.globalAlpha = 1;
  hit = markers;
  // The markers moved under a still pointer: say what is there now.
  if (pointer) hover(pointer.x, pointer.y);

  if (world.bot) {
    const x = sx(world.bot[0]), y = sy(world.bot[2]);
    ctx.fillStyle = "#182832";
    ctx.beginPath();
    ctx.arc(x, y, 10, 0, 2 * Math.PI);
    ctx.fill();
    ctx.strokeStyle = "#a9e2ff"; ctx.lineWidth = 2; ctx.stroke();
    // Facing: yaw 0 is north, clockwise.
    const a = (world.yaw * Math.PI) / 180;
    ctx.save(); ctx.translate(x, y); ctx.rotate(a); ctx.fillStyle = "#a9e2ff";
    ctx.beginPath();
    ctx.moveTo(0, -8); ctx.lineTo(5, 5); ctx.lineTo(0, 3); ctx.lineTo(-5, 5); ctx.closePath(); ctx.fill(); ctx.restore();
    label("bot", x + 14, y + 4);
  } else {
    ctx.fillStyle = colour("dim");
    ctx.fillText(selectedAtlas ? "Remembered terrain · no live observations" : "The bot is not in the world.", 12, 20);
  }
  ctx.fillStyle = "#d3dfdf"; ctx.font = "12px system-ui";
  ctx.fillText("N ↑", w - 36, 24);
  const metres = view.scale >= 3 ? 10 : 50, bar = metres * view.scale;
  ctx.strokeStyle = "#b4c6c9"; ctx.lineWidth = 1.5;
  ctx.beginPath(); ctx.moveTo(w - 16 - bar, h - 26); ctx.lineTo(w - 16 - bar, h - 21);
  ctx.lineTo(w - 16, h - 21); ctx.lineTo(w - 16, h - 26); ctx.stroke();
  ctx.font = "10px system-ui"; ctx.fillText(`${metres} m`, w - 16 - bar, h - 7);
}

// ---- markers ----

// How each kind looks: a shape, a colour, and what to call one or several.
const LOOK = {
  // structures (6.2)
  container: { shape: "square", colour: "#c8955a", one: "chest", many: "chests", prio: 40 },
  station: { shape: "diamond", colour: "#b08cff", one: "station", many: "stations", prio: 41 },
  bed: { shape: "bar", colour: "#6fa8ff", one: "bed", many: "beds", prio: 42 },
  portal: { shape: "ring", colour: "#d07cff", one: "portal", many: "portals", prio: 39 },
  fire: { shape: "circle", colour: "#ff8a3d", one: "fire", many: "fires", prio: 43 },
  smelter: { shape: "square", colour: "#ff5f3d", one: "smelter", many: "smelters", prio: 43 },
  door: { shape: "door", colour: "#a7b4c2", one: "door", many: "doors", prio: 45 },
  other: { shape: "square", colour: "#a7b4c2", one: "structure", many: "structures", prio: 46 },
  // resources (6.2)
  drop: { shape: "diamond", colour: "#e8e2a0", one: "drop", many: "drops", prio: 50, size: 4 },
  tree: { shape: "triangle", colour: "#3fa45b", one: "tree", many: "trees", prio: 60 },
  log: { shape: "bar", colour: "#a07a4a", one: "log", many: "logs", prio: 61 },
  stump: { shape: "dot", colour: "#8a6a45", one: "stump", many: "stumps", prio: 63 },
  rock: { shape: "circle", colour: "#9aa3ad", one: "rock", many: "rocks", prio: 62 },
  pickable: { shape: "dot", colour: "#e07bb5", one: "pickable", many: "pickables", prio: 64 },
};

// How close on screen, in pixels, things of one kind join one marker.
const JOIN = 14;

let hit = []; // the markers last drawn, for hovering

function collect(sx, sy) {
  const out = [];
  // A thing joins a group of its kind when it is near any member, so a
  // row of chests is one group however long. Groups it bridges merge.
  const add = (key, look, thing, x, y, extra = {}) => {
    if (x < -30 || y < -30 || x > canvas.clientWidth + 30 || y > canvas.clientHeight + 30) return;
    const near = out.filter((m) => m.key === key && m.pts.some(([px, py]) => Math.hypot(px - x, py - y) < JOIN));
    if (near.length === 0) {
      out.push({ key, look, items: [thing], pts: [[x, y]], x, y, prio: look.prio, ...extra });
      return;
    }
    // A group shows the ring and takes the label rank of its most
    // pressing member.
    const [m, ...rest] = near;
    m.items.push(thing);
    m.pts.push([x, y]);
    m.ring = m.ring || extra.ring;
    m.prio = Math.min(m.prio, look.prio);
    for (const r of rest) {
      m.items.push(...r.items);
      m.pts.push(...r.pts);
      m.ring = m.ring || r.ring;
      m.prio = Math.min(m.prio, r.prio);
      out.splice(out.indexOf(r), 1);
    }
  };
  for (const r of layers.resources ? world.resources : []) {
    const look = LOOK[r.kind] || LOOK.pickable;
    add("r:" + r.kind, look, { what: look.one, ...r }, sx(r.pos[0]), sy(r.pos[2]));
  }
  for (const s of world.structures) {
    const look = LOOK[s.kind] || LOOK.other;
    // An open door, a claimed bed and a named portal stand alone.
    const key = "s:" + s.kind + (s.open ? ":open" : "") + (s.mine ? ":mine" : "") + (s.tag ? ":" + s.tag : "");
    add(key, look, { what: look.one, ...s }, sx(s.pos[0]), sy(s.pos[2]));
  }
  if (layers.memory) {
    const live = new Set([...world.structures, ...world.resources].map((s) => s.id));
    for (const s of world.landmarks) {
      if (live.has(s.entity) || (!layers.resources && ["pickable", "rock"].includes(s.kind))) continue;
      const look = LOOK[s.kind] || LOOK.other;
      add("memory:" + s.kind, { ...look, prio: look.prio + 40 }, { ...s, what: look.one, remembered: true },
        sx(s.pos[0]), sy(s.pos[2]), { remembered: true });
    }
  }
  for (const c of world.characters) {
    const kind = c.kind === "player" ? "player" : c.kind === "tamed" ? "tamed" : c.hostile ? "hostile" : "other";
    const look = { shape: "circle", colour: colour(kind), size: c.kind === "player" ? 6 : 4,
      one: c.kind === "player" ? "player" : c.kind === "tamed" ? "tamed" : c.hostile ? "hostile" : "creature",
      prio: c.kind === "player" ? 10 : c.targetingMe ? 11 : c.hostile ? 12 : 30 };
    // Each player stands alone; creatures of one prefab group.
    add(c.kind === "player" ? "p:" + c.id : "c:" + kind + ":" + (c.prefab || ""), look, { what: look.one, ...c },
      sx(c.pos[0]), sy(c.pos[2]), { ring: c.targetingMe });
  }
  if (world.home) {
    out.push({ key: "home", look: { shape: "square", colour: colour("home"), size: 6, one: "home" },
      items: [{ what: "home", pos: world.home }], x: sx(world.home[0]), y: sy(world.home[2]), prio: 20, label: "home" });
  }
  for (const p of world.waypoints) {
    out.push({ key: "waypoint:" + p.name, look: { shape: "hollow-diamond", colour: "#d4a6ff", size: 7, one: "place" },
      items: [{ ...p, what: p.name, detail: `${p.kind} · ${p.radius ? p.radius + " m zone" : "point"} · ${p.source === "bot" ? "automatic" : "owner"}` }],
      x: sx(p.pos[0]), y: sy(p.pos[2]), prio: 19, label: p.name });
  }
  for (const c of world.left) {
    const what = Object.entries(c.items).sort((a, b) => b[1] - a[1]).map(([p, n]) => `${n} ${plain(p)}`).join(", ");
    out.push({ key: "left", look: { shape: "hollow-diamond", colour: colour("home"), size: 6, one: "left behind" },
      items: [{ what: "left behind", pos: c.pos, detail: what }], x: sx(c.pos[0]), y: sy(c.pos[2]), prio: 21, label: `left: ${what}` });
  }
  for (const m of out) {
    if (m.label) continue;
    const first = m.items[0];
    m.x = m.items.reduce((a, i) => a + sx(i.pos[0]), 0) / m.items.length;
    m.y = m.items.reduce((a, i) => a + sy(i.pos[2]), 0) / m.items.length;
    m.label = markerLabel(m, first);
  }
  return out;
}

// A marker's label: a player's name, a group's count, a drop's stack, a
// structure's kind. Single trees, rocks and the like go unlabelled; hover
// says what they are.
function markerLabel(m, first) {
  const n = m.items.length;
  if (first.kind === "player") return first.name;
  if (m.key.startsWith("c:")) {
    const stars = first.stars ? " " + "★".repeat(first.stars) : "";
    const name = plain(first.prefab || first.name || "creature") + stars;
    return n > 1 ? `${n} × ${name}` : name;
  }
  if (first.kind === "drop") {
    const total = m.items.reduce((a, i) => a + (i.stack || 1), 0);
    const kinds = new Set(m.items.map((i) => i.prefab));
    return kinds.size === 1 ? `${plain(first.prefab)} ×${total}` : `${n} drops`;
  }
  if (m.key.startsWith("r:")) return n > 1 ? `${n} ${m.look.many}` : "";
  let name = n > 1 ? `${n} ${m.look.many}` : m.look.one;
  if (first.kind === "portal" && first.tag) name += ` "${first.tag}"`;
  if (first.kind === "bed" && first.mine) name += " (mine)";
  if (first.kind === "door") name += first.open ? " (open)" : "";
  return name;
}

function drawMarker(m) {
  const { shape, colour: c } = m.look;
  const r = (m.look.size || 5) + (m.items.length > 1 ? 1 : 0);
  const { x, y } = m;
  ctx.fillStyle = ctx.strokeStyle = c;
  ctx.lineWidth = 2;
  ctx.beginPath();
  switch (shape) {
    case "square":
      ctx.rect(x - r, y - r, 2 * r, 2 * r);
      ctx.fill();
      break;
    case "diamond":
    case "hollow-diamond":
      ctx.moveTo(x, y - r - 1);
      ctx.lineTo(x + r + 1, y);
      ctx.lineTo(x, y + r + 1);
      ctx.lineTo(x - r - 1, y);
      ctx.closePath();
      shape === "diamond" ? ctx.fill() : ctx.stroke();
      break;
    case "triangle":
      ctx.moveTo(x, y - r - 1);
      ctx.lineTo(x + r, y + r);
      ctx.lineTo(x - r, y + r);
      ctx.closePath();
      ctx.fill();
      break;
    case "bar":
      ctx.rect(x - r - 1, y - r / 2, 2 * r + 2, r);
      ctx.fill();
      break;
    case "ring":
      ctx.arc(x, y, r, 0, 2 * Math.PI);
      ctx.stroke();
      break;
    case "dot":
      ctx.arc(x, y, Math.max(2.5, r - 2), 0, 2 * Math.PI);
      ctx.fill();
      break;
    case "door": {
      const open = m.items[0].open;
      ctx.rect(x - 3, y - r - 1, 6, 2 * r + 2);
      open ? ctx.stroke() : ctx.fill();
      break;
    }
    default:
      ctx.arc(x, y, r, 0, 2 * Math.PI);
      ctx.fill();
  }
  if (m.ring) {
    ctx.strokeStyle = colour("hostile");
    ctx.beginPath();
    ctx.arc(x, y, r + 4, 0, 2 * Math.PI);
    ctx.stroke();
  }
  m.r = r + 2;
}

const box = (x, y, w, h) => ({ x, y, w, h });
const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

// place puts m's label to its right, or else its left, above or below,
// wherever it overlaps no label or marker already placed.
function place(m, taken) {
  ctx.font = "12px system-ui, sans-serif";
  const w = ctx.measureText(m.label).width, h = 13, gap = m.r + 3;
  const tries = [
    box(m.x + gap, m.y - h / 2, w, h),
    box(m.x - gap - w, m.y - h / 2, w, h),
    box(m.x - w / 2, m.y - gap - h, w, h),
    box(m.x - w / 2, m.y + gap, w, h),
    box(m.x + gap, m.y - gap - h, w, h),
    box(m.x + gap, m.y + gap, w, h),
    box(m.x - gap - w, m.y - gap - h, w, h),
    box(m.x - gap - w, m.y + gap, w, h),
  ];
  const spot = tries.find((b) => !taken.some((t) => overlaps(b, t)));
  if (!spot) return;
  taken.push(spot);
  label(m.label, spot.x, spot.y + h - 2, m.prio >= 50 ? "dim" : "text");
}

function label(text, x, y, tone = "text") {
  ctx.fillStyle = colour(tone);
  ctx.font = "12px system-ui, sans-serif";
  ctx.strokeStyle = "#182126"; ctx.lineWidth = 3; ctx.strokeText(text, x, y); ctx.fillText(text, x, y);
}

function grid(w, h, sx, sy) {
  // Lines every 10 m, or 50 m when zoomed out.
  const step = view.scale < 2 ? 50 : 10;
  const left = view.centre[0] - w / 2 / view.scale, right = view.centre[0] + w / 2 / view.scale;
  const bottom = view.centre[1] - h / 2 / view.scale, top = view.centre[1] + h / 2 / view.scale;
  ctx.strokeStyle = "#c3d0d015";
  ctx.lineWidth = 1;
  ctx.beginPath();
  for (let x = Math.ceil(left / step) * step; x <= right; x += step) {
    ctx.moveTo(sx(x), 0);
    ctx.lineTo(sx(x), h);
  }
  for (let z = Math.ceil(bottom / step) * step; z <= top; z += step) {
    ctx.moveTo(0, sy(z));
    ctx.lineTo(w, sy(z));
  }
  ctx.stroke();
  ctx.fillStyle = colour("dim");
  ctx.font = "11px system-ui, sans-serif";
  ctx.fillText(`grid ${step} m · N ↑`, 8, h - 8);
}

function zoom(f) {
  view.zoomed = true;
  view.scale = Math.min(40, Math.max(0.25, view.scale * f));
  mapFetchedAt = 0; loadMap();
  draw();
}

$("zoom-in").addEventListener("click", () => zoom(1.5));
$("zoom-out").addEventListener("click", () => zoom(1 / 1.5));
canvas.addEventListener("wheel", (e) => {
  e.preventDefault();
  zoom(e.deltaY < 0 ? 1.2 : 1 / 1.2);
}, { passive: false });
$("follow-bot").addEventListener("change", (e) => {
  view.followBot = e.target.checked;
  draw();
});

// Dragging pans, and stops following the bot.
let drag = null;
canvas.addEventListener("pointerdown", (e) => {
  if (placePicking) { pickPlace(e); return; }
  drag = { x: e.clientX, y: e.clientY, centre: view.centre.slice() };
  canvas.setPointerCapture(e.pointerId);
});
canvas.addEventListener("pointermove", (e) => {
  if (!drag) return;
  const dx = (e.clientX - drag.x) / view.scale, dz = (e.clientY - drag.y) / view.scale;
  if (Math.abs(dx) + Math.abs(dz) > 0.5 && view.followBot) {
    view.followBot = false;
    $("follow-bot").checked = false;
  }
  view.centre = [drag.centre[0] - dx, drag.centre[1] + dz];
  draw();
});
canvas.addEventListener("pointerup", (e) => {
  if (drag && Math.hypot(e.clientX - drag.x, e.clientY - drag.y) < 5) {
    const rect = canvas.getBoundingClientRect(), x = e.clientX - rect.left, y = e.clientY - rect.top;
    const marker = hit.filter(m => m.key.startsWith("waypoint:") && Math.hypot(m.x-x, m.y-y) < 12)
      .sort((a,b) => Math.hypot(a.x-x,a.y-y)-Math.hypot(b.x-x,b.y-y))[0];
    if (marker) editPlace(world.waypoints.find(p => "waypoint:" + p.name === marker.key));
  }
  drag = null;
  mapFetchedAt = 0; loadMap();
});

// Hovering a marker says what it is: every item of a group.
const tip = $("map-tip");
let pointer = null; // where the pointer rests over the map, while not dragging
canvas.addEventListener("pointermove", (e) => {
  if (drag) {
    pointer = null;
    tip.hidden = true;
    return;
  }
  const rect = canvas.getBoundingClientRect();
  pointer = { x: e.clientX - rect.left, y: e.clientY - rect.top };
  hover(pointer.x, pointer.y);
});
canvas.addEventListener("pointerleave", () => {
  pointer = null;
  tip.hidden = true;
});

function hover(x, y) {
  const rect = canvas.getBoundingClientRect();
  let best = null, bestD = Infinity;
  for (const m of hit) {
    const d = Math.hypot(m.x - x, m.y - y);
    if (d <= m.r + 4 && d < bestD) (best = m), (bestD = d);
  }
  if (!best) {
    const wx = view.centre[0] + (x - rect.width / 2) / view.scale;
    const wz = view.centre[1] - (y - rect.height / 2) / view.scale;
    const s = RenkiTerrain.sample(world.tiles, wx, wz);
    const text = s ? `${plain(s.biome || "Biome unknown")} · terrain ${s.height.toFixed(1)} m${s.water > s.height ? ` · water depth ${(s.water - s.height).toFixed(1)} m` : " · above water level"}` : "Unknown terrain · no retained sample";
    const danger = layers.danger ? world.danger.filter((z) => Math.hypot(wx-z.pos[0], wz-z.pos[2]) <= z.radius) : [];
    const age = s?.observedAt ? `${s.stale ? "Stale" : "Remembered"} · oldest corner ${new Date(s.observedAt).toLocaleString()} · source ${s.sources.join(", ")} · 4 m spacing` : s ? "Session terrain · measurement age unavailable" : "";
    const lines = [text, ...(age ? [age] : []), ...danger.slice(0, 3).map((z) => `Remembered danger · ${z.hits} injuries · ${z.deaths} deaths · score ${z.score.toFixed(2)} · height ${z.pos[1].toFixed(1)} m · ${z.prefab ? plain(z.prefab) : z.by || "cause unknown"} · last hurt ${new Date(z.at).toLocaleString()}`)];
    tip.replaceChildren(...lines.map((line) => Object.assign(document.createElement("div"), { textContent: line })));
  } else {
    tip.replaceChildren(...describe(best).map((t) => Object.assign(document.createElement("div"), { textContent: t })));
  }
  tip.hidden = false;
  // Beside the pointer, kept inside the map.
  const left = Math.min(x + 14, rect.width - tip.offsetWidth - 4);
  const top = Math.min(y + 14, rect.height - tip.offsetHeight - 4);
  tip.style.left = canvas.offsetLeft + Math.max(4, left) + "px";
  tip.style.top = canvas.offsetTop + Math.max(4, top) + "px";
}

// describe lists a marker's items: what each is and how far from the bot,
// nearest first, at most a dozen.
function describe(m) {
  const far = (p) => (world.bot ? Math.hypot(p[0] - world.bot[0], p[2] - world.bot[2]) : null);
  const items = [...m.items].sort((a, b) => (far(a.pos) ?? 0) - (far(b.pos) ?? 0));
  const chests = items.every((i) => i.kind === "container");
  const limit = chests && items.length > 1 ? 3 : 12;
  const lines = items.slice(0, limit).flatMap((i) => {
    const parts = [i.kind === "player" ? i.name : i.what];
    if (i.kind === "container") {
      parts.push(i.id || i.entity);
      parts.push(`x ${i.pos[0].toFixed(1)}, z ${i.pos[2].toFixed(1)}`);
    } else if (i.prefab) parts.push(plain(i.prefab));
    if (i.detail) parts.push(i.detail);
    if (i.stack > 1) parts.push(`×${i.stack}`);
    if (i.kind === "door" && !i.remembered) parts.push(i.open ? "open" : "shut");
    if (i.remembered) parts.push(`last seen ${new Date(i.at).toLocaleString()}`);
    if (i.mine) parts.push("the bot's");
    if (i.tag) parts.push(`"${i.tag}"`);
    if (i.minTool != null) parts.push(`needs tool tier ${i.minTool}`);
    if (i.stars) parts.push("★".repeat(i.stars));
    if (typeof i.hpFrac === "number") parts.push(`${Math.round(i.hpFrac * 100)}% health`);
    if (i.targetingMe) parts.push("after the bot");
    const d = far(i.pos);
    if (d != null) parts.push(`${Math.round(d)} m`);
    if (i.kind !== "container") return [parts.join(" · ")];
    const chest = world.containers.get(i.id || i.entity);
    if (!chest) return [parts.join(" · "), "Contents unknown · Tas has not opened this chest"];
    const seen = new Date(chest.at).toLocaleString();
    const totals = new Map();
    for (const item of chest.items || []) totals.set(item.prefab, (totals.get(item.prefab) || 0) + item.stack);
    const contents = [...totals].sort(([a], [b]) => a.localeCompare(b));
    const details = contents.slice(0, items.length > 1 ? 4 : 12).map(([prefab, count]) => `${count} × ${plain(prefab)}`);
    const extra = contents.length - details.length;
    if (extra) details.push(`+${extra} more item types`);
    const provenance = chest.observedEntity && chest.observedEntity !== chest.entity
      ? `Contents from a previous object here · last opened ${seen} · needs checking`
      : `Last opened ${seen}`;
    return [parts.join(" · "), provenance, details.length ? details.join(" · ") : "Empty when last opened"];
  });
  if (items.length > limit) lines.push(`and ${items.length - limit} more${chests ? " chests · zoom in for individual contents" : ""}`);
  return lines;
}
window.addEventListener("resize", draw);

// ---- buttons ----

let acting = false;
let resultTimer = 0;

const CONFIRM = {
  logout: "Log the bot's character out of the game? It will need the VM to log back in.",
};

async function act(action) {
  if (CONFIRM[action] && !confirm(CONFIRM[action])) return;
  const out = $("act-result");
  const buttons = document.querySelectorAll("button.act");
  acting = true;
  buttons.forEach((b) => (b.disabled = true));
  let text, ok;
  try {
    const res = await fetch(`/api/act/${action}`, { method: "POST", headers: { "X-Renki-Act": "1" } });
    if (res.status === 401) return void (location.href = "/login");
    const body = await res.json().catch(() => ({}));
    ok = res.ok;
    text = body.text || res.statusText;
  } catch (e) {
    ok = false;
    text = "the controller did not answer";
  } finally {
    acting = false;
  }
  // The next status re-enables the buttons if the bot is still connected.
  out.textContent = text;
  out.className = ok ? "" : "error";
  clearTimeout(resultTimer);
  resultTimer = setTimeout(() => (out.textContent = ""), 8000);
}

for (const b of document.querySelectorAll("button.act")) b.addEventListener("click", () => act(b.dataset.act));

// ---- voice ----

// The model the voice uses now, what the backend offers, and whether the
// user has picked one in the list that is not yet in use.
let model = { voice: false, name: "" };
let offered = [];
let picked = false;
let modelError = "";
let modelTimer = 0;

function onModel(s) {
  const changed = s.model?.name !== model.name;
  model = { voice: !!s.voice, name: s.model?.name || "" };
  $("voice-summary").textContent = model.voice ? model.name || "terva's default" : "off: commands only";
  if (changed) {
    picked = false;
    fillModels();
    loadModelNote();
  }
}

function fillModels() {
  const sel = $("model-select");
  const keep = picked ? sel.value : model.name;
  const names = [...new Set([model.name, ...offered].filter(Boolean))].sort();
  // Rebuilt when the names or the one in use change, so the mark follows.
  const key = model.name + "\n" + names.join("\n");
  if (sel.dataset.names !== key) {
    sel.dataset.names = key;
    sel.replaceChildren(...names.map((n) => new Option(n === model.name ? `${n} (in use)` : n, n)));
  }
  sel.value = keep;
  if (sel.value !== keep) {
    picked = false; // the pick is no longer offered
    sel.value = model.name;
  }
}

async function loadModels() {
  const res = await fetch("/api/models").catch(() => null);
  const body = res && res.ok ? await res.json() : { error: "the controller did not answer" };
  offered = body.models || [];
  modelError = body.error ? `Cannot list the backend's models: ${body.error}` : "";
  fillModels();
  loadModelNote();
}

// Who chose the model in use, from the settings store.
async function loadModelNote() {
  const res = await fetch("/api/settings").catch(() => null);
  const entries = res && res.ok ? await res.json() : [];
  const e = entries.find((x) => x.key === "model.name");
  let who = "";
  if (e?.set) who = `Chosen by ${e.set.by}, ${new Date(e.set.at).toLocaleString()}.`;
  else if (e) who = "As the config file has it.";
  $("model-note").textContent = [who, modelError].filter(Boolean).join(" ");
}

async function useModel() {
  const name = $("model-select").value;
  const out = $("model-result");
  if (!name || name === model.name) return;
  const res = await fetch("/api/setting", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Renki-Act": "1" },
    body: JSON.stringify({ key: "model.name", value: name }),
  }).catch(() => null);
  const body = res ? await res.json().catch(() => ({})) : {};
  out.className = res?.ok ? "" : "error";
  out.textContent = res?.ok ? "The next reply uses it." : body.text || "the controller did not answer";
  if (res?.ok) {
    picked = false;
    model.name = ""; // the next status brings the new one and the note
  }
  clearTimeout(modelTimer);
  modelTimer = setTimeout(() => (out.textContent = ""), 8000);
}

$("model-select").addEventListener("change", () => (picked = true));
$("model-use").addEventListener("click", useModel);
$("model-refresh").addEventListener("click", loadModels);
$("voice").addEventListener("toggle", () => $("voice").open && loadModels());

// ---- settings ----

// What each runtime setting is, for the page (internal/settings Keys).
const SETTING = {
  owners: "Owners, who give orders (comma-separated)",
  nicknames: "Nicknames it also answers to",
  allowedTasks: "Tasks the bot may be sent",
  voice: "LLM replies in chat",
  "model.provider": "Voice: terva provider",
  "model.name": "Voice: model",
  "model.baseURL": "Voice: backend URL",
  "home.whenLost": "Go home when it loses its leader",
  "home.whenStuck": "Go home when stuck",
  "home.afterOutage": "Go home after an outage",
  "home.idleMinutes": "Go home after idling this many minutes (0: never)",
  "home.afterDeath": "After a death",
  "overload.drop": "Drop things when carrying too much",
 "presence.sharePosition": "Share map marker with the server",
 "presence.marcoPolo": "Marco/Polo routines",
 "players.profiles": "Player control profiles",
};
const AFTER_DEATH = ["home", "resume", "stay"];
// Lists, which come as null when empty.
const LISTS = new Set(["owners", "nicknames", "allowedTasks"]);

// fresh is the key just changed from this page, whose row is rebuilt
// even while it has the focus.
async function loadSettings(fresh) {
  const res = await fetch("/api/settings").catch(() => null);
  if (!res || !res.ok) return;
  const entries = await res.json();
  // A row being edited stays as it is, so a refresh does not throw the
  // edit away; the rest show the values now.
  const a = document.activeElement;
  const typing = a?.matches?.('#settings-table input[type="text"], #settings-table input[type="number"], #settings-table select');
  const editing = typing ? a.closest("tr") : null;
  const table = $("settings-table");
  const old = [...table.querySelectorAll("tr")];
  if (!editing || old.length !== entries.length) {
    table.replaceChildren(...entries.map(settingRow));
  } else {
    // Swapped one by one, so the row being typed in is never detached and
    // keeps the focus.
    entries.forEach((e, i) => {
      if (old[i] !== editing || e.key === fresh) old[i].replaceWith(settingRow(e));
    });
  }
  const changed = entries.filter((e) => e.set).length;
  $("settings-summary").textContent = changed ? `${changed} changed from the config` : "as the config has them";
}

function settingRow(e) {
  const tr = document.createElement("tr");
  tr.dataset.key = e.key;
  if (e.set) tr.className = "changed";
  const key = document.createElement("td");
  key.className = "key";
  key.textContent = SETTING[e.key] || e.key;
  key.title = e.key;
  const cell = document.createElement("td");
  cell.append(settingInput(e));
  const src = document.createElement("td");
  src.className = "src";
  src.textContent = e.set ? `${e.set.by}, ${new Date(e.set.at).toLocaleString()}` : "config";
  if (e.pin) src.textContent += " · pinned";
  src.title = `config: ${JSON.stringify(e.config)}`;
  const acts = document.createElement("td");
  if (e.set) acts.append(button("Reset", "Back to the config's value", () => changeSetting({ key: e.key, reset: true })));
  acts.append(
    button(e.pin ? "Unpin" : "Pin", e.pin ? "Let the agent change it" : "Keep the agent from changing it", () =>
      changeSetting({ key: e.key, pin: !e.pin }),
    ),
  );
  tr.append(key, cell, src, acts);
  return tr;
}

function button(text, title, onClick) {
  const b = document.createElement("button");
  b.textContent = text;
  b.title = title;
  b.addEventListener("click", onClick);
  return b;
}

function settingInput(e) {
  const v = e.value;
  const send = (value) => changeSetting({ key: e.key, value });
  let el;
  if (e.key === "players.profiles") {
    el = document.createElement("span"); el.textContent = "Edit in Players & presence above.";
  } else if (typeof v === "boolean") {
    el = document.createElement("input");
    el.type = "checkbox";
    el.checked = v;
    el.addEventListener("change", () => send(el.checked));
  } else if (typeof v === "number") {
    el = document.createElement("input");
    el.type = "number";
    el.min = "0";
    el.step = "any";
    el.value = v;
    el.addEventListener("change", () => {
      // An empty or unreadable box is not zero, which would turn a rule off.
      if (el.value.trim() === "" || !Number.isFinite(Number(el.value))) {
        el.value = v;
        settingResult(false, "That is not a number.");
        return;
      }
      send(Number(el.value));
    });
  } else if (e.key === "home.afterDeath") {
    el = document.createElement("select");
    el.append(...AFTER_DEATH.map((o) => new Option(o, o)));
    el.value = v;
    el.addEventListener("change", () => send(el.value));
  } else if (Array.isArray(v) || LISTS.has(e.key)) {
    el = document.createElement("input");
    el.type = "text";
    el.value = (v || []).join(", ");
    el.addEventListener("change", () =>
      send(el.value.split(",").map((x) => x.trim()).filter(Boolean)),
    );
  } else {
    el = document.createElement("input");
    el.type = "text";
    el.value = v ?? "";
    el.addEventListener("change", () => send(el.value.trim()));
  }
  el.setAttribute("aria-label", SETTING[e.key] || e.key);
  return el;
}

let settingsTimer = 0;

async function changeSetting(body) {
  const res = await fetch("/api/setting", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Renki-Act": "1" },
    body: JSON.stringify(body),
  }).catch(() => null);
  const answer = res ? await res.json().catch(() => ({})) : {};
  settingResult(!!res?.ok, answer.text || "the controller did not answer");
  loadSettings(body.key); // the value now, which a refused change leaves as it was
}

function settingResult(ok, text) {
  const out = $("settings-result");
  out.className = ok ? "" : "error";
  out.textContent = text;
  clearTimeout(settingsTimer);
  settingsTimer = setTimeout(() => (out.textContent = ""), 8000);
}

$("settings").addEventListener("toggle", () => $("settings").open && loadSettings());

// ---- the stream ----

function connect() {
  const es = new EventSource("/api/events");
  es.onopen = () => { mapGeneration++; mapFetchedAt = 0; };
  es.addEventListener("record", (e) => onRecord(JSON.parse(e.data)));
  es.addEventListener("status", (e) => onStatus(JSON.parse(e.data)));
  es.onerror = async () => {
    chip("link", "reconnecting…", "off");
    // EventSource retries by itself, unless the login has gone.
    const res = await fetch("/api/status").catch(() => null);
    if (res && res.status === 401) location.href = "/login";
  };
}

connect();
loadDays();
draw();

// ---- named places and circular zones ----
let placeOriginal = "", placeExpected = null, placePicking = false;
function placeResult(text, error = false) {
  $("place-result").textContent = text;
  $("place-result").classList.toggle("error", error);
}
function syncPlaceList() {
  const list = $("place-list");
  list.replaceChildren(Object.assign(document.createElement("option"), { value: "", textContent: "Choose a place" }));
  for (const p of world.waypoints) list.append(Object.assign(document.createElement("option"), { value: p.name,
    textContent: `${p.name} · ${p.kind}${p.source === "bot" ? " · automatic" : ""}` }));
  list.value = placeOriginal;
  $("place-count").textContent = world.waypoints.length ? `(${world.waypoints.length})` : "";
}
function resetPlaceEditor() {
  placeOriginal = ""; placeExpected = null; placePicking = false;
  $("place-form").reset(); $("place-nearby").replaceChildren();
  $("place-source").textContent = ""; placeResult("");
  $("place-pick").setAttribute("aria-pressed", "false");
  $("map").classList.remove("picking-place"); syncPlaceList();
  for (const id of ["place-go", "place-delete", "place-near"]) $(id).disabled = true;
}
function editPlace(p) {
  if (!p) return;
  placeOriginal = p.name; placeExpected = p.at;
  $("places").open = true; $("place-list").value = p.name;
  $("place-name").value = p.name; $("place-kind").value = p.kind;
  ["x", "y", "z"].forEach((key, i) => $("place-" + key).value = p.pos[i].toFixed(1));
  $("place-radius").value = p.radius;
  $("place-source").textContent = `${p.source === "bot" ? "Automatic" : "Set by " + p.setBy} · ${new Date(p.at).toLocaleString()}${p.evidence ? " · " + p.evidence : ""}`;
  $("place-nearby").replaceChildren(); placeResult("");
  for (const id of ["place-go", "place-delete", "place-near"]) $(id).disabled = false;
}
function useBotPlace() {
  if (!world.bot) { placeResult("Tas must be in the world to use his position.", true); return; }
  ["x", "y", "z"].forEach((key, i) => $("place-" + key).value = world.bot[i].toFixed(1));
}
function pickPlace(e) {
  const rect = canvas.getBoundingClientRect();
  const x = view.centre[0] + (e.clientX - rect.left - rect.width / 2) / view.scale;
  const z = view.centre[1] - (e.clientY - rect.top - rect.height / 2) / view.scale;
  if ($("place-ground").checked) {
    const sample = RenkiTerrain.sample(world.tiles, x, z);
    if (!sample) { placeResult("Terrain height is unknown here. Set the height explicitly or use Tas’s position.", true); return; }
    $("place-y").value = sample.height.toFixed(1);
  }
  $("place-x").value = x.toFixed(1); $("place-z").value = z.toFixed(1);
  placePicking = false; canvas.classList.remove("picking-place"); $("place-pick").setAttribute("aria-pressed", "false");
  placeResult("Position selected. Save to keep it.");
}
async function writePlace(action) {
  if (!mapWorld) { placeResult("Tas must join a world before editing its places.", true); return; }
  const selectedWorld = mapWorld, generation = mapGeneration;
  const command = { action, world: selectedWorld, name: placeOriginal, expectedAt: placeExpected };
  if (action === "set") command.waypoint = { name: $("place-name").value, kind: $("place-kind").value,
    pos: ["x", "y", "z"].map(key => Number($("place-" + key).value)), radius: Number($("place-radius").value) };
  try {
    const response = await fetch("/api/waypoints", { method: "POST", headers: { "Content-Type": "application/json", "X-Renki-Act": "1" }, body: JSON.stringify(command) });
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    if (mapWorld !== selectedWorld || mapGeneration !== generation) return;
    world.waypoints = data.waypoints || []; syncPlaceList(); draw();
    if (action === "set") {
      const name = command.waypoint.name.trim().toLowerCase().replace(/[-_]/g, " ").replace(/\s+/g, " ").replace(/^(?:the )+/, "");
      editPlace(world.waypoints.find(p => p.name === name)); placeResult("Saved.");
    } else if (action === "delete") { resetPlaceEditor(); placeResult("Deleted. Automatic discovery will not recreate this purpose."); }
    else placeResult("Heading to the saved place.");
  } catch (error) { if (mapWorld === selectedWorld && mapGeneration === generation) placeResult(error.message, true); }
}
$("place-list").addEventListener("change", e => editPlace(world.waypoints.find(p => p.name === e.target.value)));
$("place-new").addEventListener("click", () => { resetPlaceEditor(); $("places").open = true; useBotPlace(); $("place-name").focus(); });
$("place-bot").addEventListener("click", useBotPlace);
$("place-pick").addEventListener("click", () => {
  placePicking = !placePicking; canvas.classList.toggle("picking-place", placePicking);
  $("place-pick").setAttribute("aria-pressed", String(placePicking));
  if (placePicking) canvas.scrollIntoView({block: "nearest"});
  placeResult(placePicking ? "Click the map to choose a position, then save." : "");
});
$("place-form").addEventListener("submit", e => { e.preventDefault(); writePlace("set"); });
$("place-delete").addEventListener("click", () => writePlace("delete"));
$("place-go").addEventListener("click", () => writePlace("go"));
$("place-near").addEventListener("click", async () => {
  const name = placeOriginal, selectedWorld = mapWorld, generation = mapGeneration;
  try {
    const response = await fetch("/api/waypoints?" + new URLSearchParams({world: selectedWorld, name}), {cache: "no-store"});
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    if (generation !== mapGeneration || name !== placeOriginal) return;
    const lines = (data.nearby || []).map(n => {
      const distance = world.waypoints.find(p => p.name === name)?.pos;
      const metres = distance ? Math.round(Math.hypot(n.pos[0]-distance[0],n.pos[2]-distance[2])) + " m · " : "";
      let text = `${metres}${plain(n.prefab || n.kind)} · ${n.source === "current" ? "currently seen" : "last seen " + new Date(n.at).toLocaleString()}`;
      if (n.kind === "container") {
        const counts = new Map();
        for (const item of n.contents?.items || []) counts.set(item.prefab, (counts.get(item.prefab) || 0) + item.stack);
        text += n.contents ? ` · contents last checked ${new Date(n.contents.at).toLocaleString()}: ` + ([...counts].map(([p,c]) => `${c} ${plain(p)}`).join(", ") || "empty") : " · contents unknown";
      }
      return Object.assign(document.createElement("p"), {textContent: text});
    });
    $("place-nearby").replaceChildren(...(lines.length ? lines : [Object.assign(document.createElement("p"), {textContent: "No recorded observations within this zone. Coverage may be incomplete."})]));
    placeResult("Up to 24 nearby observations; this is not a complete resource survey.");
  } catch (error) { if (generation === mapGeneration) placeResult(error.message, true); }
});
resetPlaceEditor();

// ---- persistent objectives ----
let goalSnapshot = null, goalRenderKey = "";
function renderGoals(s) {
  goalSnapshot = s;
  const active = s.connected && s.phase === "in_world" && !!s.pos;
  $("goal-combat").disabled = !active;
  const records = s.goals || [];
  const key = JSON.stringify([active, s.world, records]);
  if (key === goalRenderKey) return;
  goalRenderKey = key;
  $("goal-count").textContent = records.length ? `(${records.length})` : "";
  const list = $("goal-list"); list.replaceChildren();
  if (!records.length) { list.textContent = "No goals yet."; return; }
  for (const g of records) {
    const article = document.createElement("article"); article.className = "goal-record";
    const title = document.createElement("strong"); title.textContent = `Greyling encounter · ${g.status.replaceAll("_", " ")}`;
    const progress = document.createElement("p");
    progress.textContent = `${g.world} · Attempts ${g.attempts}/${g.maxAttempts} · Tasks ${g.taskCount}/${g.maxTasks} · Policy ${g.policyHash.slice(0, 8)}`;
    const reason = document.createElement("p"); reason.textContent = g.reason || "Objective saved";
    article.append(title, progress, reason);
    if (g.proof) { const proof = document.createElement("p"); proof.textContent = `Verified: ${g.proof.detail} (${g.proof.taskId})`; article.append(proof); }
    if (g.history?.length) {
      const history = document.createElement("ol");
      for (const outcome of g.history.slice(-3)) { const row = document.createElement("li"); row.textContent = `${outcome.status}: ${outcome.reason || outcome.detail || "no detail"} (${outcome.taskId})`; history.append(row); }
      article.append(history);
    }
    if (["ready", "suspended", "needs_plan"].includes(g.status) && !g.pending) {
      const resume = document.createElement("button"); resume.textContent = "Resume"; resume.disabled = !active || s.world !== g.world;
      resume.addEventListener("click", () => goalCommand({action:"resume", id:g.id})); article.append(resume);
    }
    if (!["succeeded", "cancelled"].includes(g.status)) {
      const stop = document.createElement("button"); stop.textContent = "Stop goal";
      stop.addEventListener("click", () => goalCommand({action:"stop", id:g.id})); article.append(stop);
    }
    list.append(article);
  }
}
async function goalCommand(body) {
  const out = $("goal-result"); out.textContent = "Saving goal…";
  try {
    const res = await fetch("/api/goals", {method:"POST", headers:{"Content-Type":"application/json", "X-Renki-Act":"1"}, body:JSON.stringify(body)});
    if (!res.ok) throw new Error(await res.text());
    const g = await res.json(); out.textContent = `Goal ${g.status.replaceAll("_", " ")}.`;
  } catch (e) { out.textContent = e.message; }
}
$("goal-combat").addEventListener("click", () => {
  const s = goalSnapshot;
  if (s?.connected && s.phase === "in_world" && s.pos) goalCommand({action:"start", spec:{kind:"combat_trial", world:s.world, center:s.pos, radius:200}});
});

// ---- player profiles and map presence ----
let presenceSnapshot = null, profileRenderKey = "";
function renderPresence(s) {
  presenceSnapshot = s;
  $("share-position").checked = !!s.presence?.sharePosition;
  $("marco-polo").checked = !!s.presence?.marcoPolo;
  const desired = !!s.presence?.sharePosition;
  $("presence-summary").textContent = `${desired ? "marker shared" : "marker hidden"} · Marco/Polo ${s.presence?.marcoPolo ? "on" : "off"}`;
  $("marker-state").textContent = !s.connected ? "Applies when Tas joins" : s.sharePosition == null ? "Waiting for marker observation" : s.sharePosition === desired ? (desired ? "Marker shared" : "Marker hidden") : "Applying marker change…";
  $("profile-owners").textContent = `Owners: ${(s.owners || []).join(", ") || "none"}. Owners also receive Marco greetings.`;
  const entries = Object.entries(s.players?.profiles || {}).sort(([a], [b]) => a.localeCompare(b));
  const key = JSON.stringify(entries); if (key === profileRenderKey) return; profileRenderKey = key;
  const table = $("profile-table"); table.replaceChildren();
  for (const [name, role] of entries) {
    const row = document.createElement("tr"), who = document.createElement("td"), cell = document.createElement("td"), actions = document.createElement("td");
    who.textContent = name;
    const choice = document.createElement("select"); choice.setAttribute("aria-label", `${name}’s profile`);
    for (const value of ["guest", "companion", "operator"]) choice.append(new Option(value, value));
    choice.value = role; choice.addEventListener("change", () => assignProfile(name, choice.value)); cell.append(choice);
    actions.append(button("Remove", `Remove ${name}’s profile`, () => assignProfile(name, null)));
    row.append(who, cell, actions); table.append(row);
  }
}
async function presenceSetting(key, value) {
  const out = $("presence-result"); out.textContent = "Saving…";
  try {
    const res = await fetch("/api/setting", {method:"POST", headers:{"Content-Type":"application/json", "X-Renki-Act":"1"}, body:JSON.stringify({key, value})});
    const answer = await res.json(); if (!res.ok) throw new Error(answer.text || "Could not save");
    out.textContent = answer.text || "Saved.";
  } catch (e) { out.textContent = e.message; profileRenderKey = ""; if (presenceSnapshot) renderPresence(presenceSnapshot); }
}
function assignProfile(name, role) {
  const profiles = {...presenceSnapshot?.players?.profiles};
  if (role === null) delete profiles[name]; else Object.defineProperty(profiles, name, {value:role, enumerable:true, configurable:true, writable:true});
  return presenceSetting("players.profiles", profiles);
}
$("share-position").addEventListener("change", () => presenceSetting("presence.sharePosition", $("share-position").checked));
$("marco-polo").addEventListener("change", () => presenceSetting("presence.marcoPolo", $("marco-polo").checked));
$("profile-form").addEventListener("submit", (event) => {
  event.preventDefault(); const name = $("profile-name").value.trim();
  if (name) assignProfile(name, $("profile-role").value);
});
