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

  if (world.home) {
    ctx.fillStyle = colour("home");
    const x = sx(world.home[0]), y = sy(world.home[2]);
    ctx.fillRect(x - 6, y - 6, 12, 12);
    label("home", x + 9, y + 4);
  }

  for (const c of world.left) {
    ctx.strokeStyle = colour("home");
    ctx.lineWidth = 2;
    const x = sx(c.pos[0]), y = sy(c.pos[2]);
    ctx.beginPath();
    ctx.moveTo(x, y - 6);
    ctx.lineTo(x + 6, y);
    ctx.lineTo(x, y + 6);
    ctx.lineTo(x - 6, y);
    ctx.closePath();
    ctx.stroke();
    const what = Object.entries(c.items).sort((a, b) => b[1] - a[1]).map(([p, n]) => `${n} ${plain(p)}`).join(", ");
    label(`left: ${what}`, x + 9, y + 4);
  }

  // Resources only once zoomed in enough to tell them apart.
  if (view.scale >= 3) {
    for (const r of world.resources) {
      const x = sx(r.pos[0]), y = sy(r.pos[2]);
      ctx.fillStyle = colour("resource");
      ctx.fillRect(x - 2, y - 2, 4, 4);
      if (view.scale >= 12) label(r.prefab, x + 5, y + 4, "dim");
    }
  }

  for (const s of world.structures) {
    const x = sx(s.pos[0]), y = sy(s.pos[2]);
    ctx.fillStyle = ctx.strokeStyle = colour("structure");
    if (s.kind === "door") {
      ctx.lineWidth = 2;
      s.open ? ctx.strokeRect(x - 3, y - 5, 6, 10) : ctx.fillRect(x - 3, y - 5, 6, 10);
    } else {
      ctx.fillRect(x - 4, y - 4, 8, 8);
    }
    const name = s.kind === "portal" && s.tag ? `portal "${s.tag}"` : s.kind === "bed" && s.mine ? "bed (mine)" : s.kind;
    if (view.scale >= 6) label(name, x + 7, y + 4, "dim");
  }

  for (const c of world.characters) {
    const x = sx(c.pos[0]), y = sy(c.pos[2]);
    const player = c.kind === "player";
    ctx.fillStyle = colour(player ? "player" : c.kind === "tamed" ? "tamed" : c.hostile ? "hostile" : "other");
    ctx.beginPath();
    ctx.arc(x, y, player ? 6 : 4, 0, 2 * Math.PI);
    ctx.fill();
    if (c.targetingMe) {
      ctx.strokeStyle = colour("hostile");
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(x, y, 8, 0, 2 * Math.PI);
      ctx.stroke();
    }
    const stars = c.stars ? " " + "★".repeat(c.stars) : "";
    if (player || c.targetingMe || view.scale >= 6) label((c.name || c.prefab || "") + stars, x + 9, y + 4);
  }

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
