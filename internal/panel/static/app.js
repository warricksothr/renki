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
  home: null,
  left: [], // what the bot left on the ground: {pos, items}
  path: null, // {from, to}
  stuck: [], // {pos, attempt}
};

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
    for (const li of feed.children) li.classList.toggle("hidden", !shown.has(li.dataset.cat));
  });
}

function addLine(ev) {
  if (ev.cat === "perception") return;
  if (ev.cat === "state") {
    const now = Date.parse(ev.at);
    if (now - lastStateLine < 1000) return;
    lastStateLine = now;
  }
  const li = document.createElement("li");
  li.textContent = ev.line;
  li.dataset.cat = ev.cat;
  li.className = ev.cat;
  if (ev.level === "WARN") li.classList.add("warn");
  if (ev.level === "ERROR") li.classList.add("error");
  if (!shown.has(ev.cat)) li.classList.add("hidden");
  const atBottom = feed.scrollTop + feed.clientHeight >= feed.scrollHeight - 20;
  feed.append(li);
  while (feed.children.length > FEED_MAX) feed.firstChild.remove();
  if (atBottom && !$("pause").checked) feed.scrollTop = feed.scrollHeight;
}

// ---- records into the map ----

function onRecord(ev) {
  addLine(ev);
  const f = ev.frame;
  if (!f) return;
  switch (f.type) {
    case "state":
      if (f.phase !== "in_world") return;
      world.bot = f.pos;
      world.yaw = f.yaw || 0;
      const last = world.trail[world.trail.length - 1];
      if (!last || Math.hypot(last[0] - f.pos[0], last[2] - f.pos[2]) > 0.3) {
        world.trail.push(f.pos);
        if (world.trail.length > TRAIL_MAX) world.trail.shift();
      }
      $("pos").textContent = `x ${f.pos[0].toFixed(1)}  z ${f.pos[2].toFixed(1)}  y ${f.pos[1].toFixed(1)}`;
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
    case "perception":
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
  chip("link", s.connected ? "connected" : "no plugin", s.connected ? "on" : "off");
  chip("who", s.character ? `${s.character}${s.world ? " in " + s.world : ""}` : "");
  chip("phase", s.phase ? s.phase.replace("_", " ") : "");
  chip("task", s.task ? `${s.task.task} (${s.task.id})` : s.connected ? "idle" : "");
  if (s.perceptionRadius) view.radius = s.perceptionRadius;
  chip("hp", s.hpMax ? `hp ${Math.round(s.hp)}/${Math.round(s.hpMax)}` : "");
  world.left = s.leftBehind || [];
  if (s.home) {
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
  if (!view.zoomed && view.radius > 0) view.scale = Math.min(w, h) / (2 * view.radius * 1.1);
  if (view.followBot && world.bot) view.centre = [world.bot[0], world.bot[2]];
  // North (+z) is up, east (+x) is right.
  const sx = (x) => w / 2 + (x - view.centre[0]) * view.scale;
  const sy = (z) => h / 2 - (z - view.centre[1]) * view.scale;

  grid(w, h, sx, sy);

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
  if (world.bot) taken.push(box(sx(world.bot[0]) - 9, sy(world.bot[2]) - 9, 18, 18));
  for (const m of markers) {
    drawMarker(m);
    taken.push(box(m.x - m.r, m.y - m.r, 2 * m.r, 2 * m.r));
  }
  for (const m of [...markers].sort((a, b) => a.prio - b.prio)) {
    if (m.label) place(m, taken);
  }
  hit = markers;
  // The markers moved under a still pointer: say what is there now.
  if (pointer) hover(pointer.x, pointer.y);

  if (world.bot) {
    const x = sx(world.bot[0]), y = sy(world.bot[2]);
    ctx.fillStyle = colour("bot");
    ctx.beginPath();
    ctx.arc(x, y, 7, 0, 2 * Math.PI);
    ctx.fill();
    // Facing: yaw 0 is north, clockwise.
    const a = (world.yaw * Math.PI) / 180;
    ctx.strokeStyle = colour("bot");
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + Math.sin(a) * 16, y - Math.cos(a) * 16);
    ctx.stroke();
  } else {
    ctx.fillStyle = colour("dim");
    ctx.fillText("The bot is not in the world.", 12, 20);
  }
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
  for (const r of world.resources) {
    const look = LOOK[r.kind] || LOOK.pickable;
    add("r:" + r.kind, look, { what: look.one, ...r }, sx(r.pos[0]), sy(r.pos[2]));
  }
  for (const s of world.structures) {
    const look = LOOK[s.kind] || LOOK.other;
    // An open door, a claimed bed and a named portal stand alone.
    const key = "s:" + s.kind + (s.open ? ":open" : "") + (s.mine ? ":mine" : "") + (s.tag ? ":" + s.tag : "");
    add(key, look, { what: look.one, ...s }, sx(s.pos[0]), sy(s.pos[2]));
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
  ctx.fillText(text, x, y);
}

function grid(w, h, sx, sy) {
  // Lines every 10 m, or 50 m when zoomed out.
  const step = view.scale < 2 ? 50 : 10;
  const left = view.centre[0] - w / 2 / view.scale, right = view.centre[0] + w / 2 / view.scale;
  const bottom = view.centre[1] - h / 2 / view.scale, top = view.centre[1] + h / 2 / view.scale;
  ctx.strokeStyle = colour("line");
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
canvas.addEventListener("pointerup", () => (drag = null));

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
    tip.hidden = true;
    return;
  }
  tip.replaceChildren(...describe(best).map((t) => Object.assign(document.createElement("div"), { textContent: t })));
  tip.hidden = false;
  // Beside the pointer, kept inside the map.
  const left = Math.min(x + 14, rect.width - tip.offsetWidth - 4);
  const top = Math.min(y + 14, rect.height - tip.offsetHeight - 4);
  tip.style.left = Math.max(4, left) + "px";
  tip.style.top = Math.max(4, top) + "px";
}

// describe lists a marker's items: what each is and how far from the bot,
// nearest first, at most a dozen.
function describe(m) {
  const far = (p) => (world.bot ? Math.hypot(p[0] - world.bot[0], p[2] - world.bot[2]) : null);
  const items = [...m.items].sort((a, b) => (far(a.pos) ?? 0) - (far(b.pos) ?? 0));
  const lines = items.slice(0, 12).map((i) => {
    const parts = [i.kind === "player" ? i.name : i.what];
    if (i.prefab) parts.push(plain(i.prefab));
    if (i.detail) parts.push(i.detail);
    if (i.stack > 1) parts.push(`×${i.stack}`);
    if (i.kind === "door") parts.push(i.open ? "open" : "shut");
    if (i.mine) parts.push("the bot's");
    if (i.tag) parts.push(`"${i.tag}"`);
    if (i.minTool) parts.push(`needs tool tier ${i.minTool}`);
    if (i.stars) parts.push("★".repeat(i.stars));
    if (typeof i.hpFrac === "number") parts.push(`${Math.round(i.hpFrac * 100)}% health`);
    if (i.targetingMe) parts.push("after the bot");
    const d = far(i.pos);
    if (d != null) parts.push(`${Math.round(d)} m`);
    return parts.join(" · ");
  });
  if (items.length > 12) lines.push(`and ${items.length - 12} more`);
  return lines;
}
window.addEventListener("resize", draw);

// ---- the stream ----

function connect() {
  const es = new EventSource("/api/events");
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
draw();
