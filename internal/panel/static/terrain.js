"use strict";

// Visual heightmap sampling: it never classifies a route as walkable.
const RenkiTerrain = (() => {
  const colours = { Meadows: "#344a36", BlackForest: "#233b30", Swamp: "#404631", Mountain: "#717c82",
    Plains: "#686344", AshLands: "#513d3b", DeepNorth: "#7a8b92", Ocean: "#304b49", Mistlands: "#41434e" };
  const key = (x, z) => `${x},${z}`;
  function sample(tiles, x, z) {
    const tx = Math.floor(x / 16), tz = Math.floor(z / 16), tile = tiles.get(key(tx, tz));
    if (!tile) return null;
    const gx = (x - tx * 16) / 4, gz = (z - tz * 16) / 4;
    const ix = Math.min(3, Math.floor(gx)), iz = Math.min(3, Math.floor(gz));
    const i = iz * 5 + ix, hs = [tile.heights[i], tile.heights[i + 1], tile.heights[i + 5], tile.heights[i + 6]];
    if (!hs.every(Number.isFinite)) return null;
    const a = gx - ix, b = gz - iz;
    return { height: (hs[0] * (1 - a) + hs[1] * a) * (1 - b) + (hs[2] * (1 - a) + hs[3] * a) * b,
      biome: tile.biomes[i], water: tile.waterLevel };
  }
  function crossing(a, b, level) {
    const t = (level - a.h) / (b.h - a.h);
    return { x: a.x + t * (b.x - a.x), z: a.z + t * (b.z - a.z), h: level };
  }
  // Clip one triangle against a water plane. Null/unobserved corners are
  // filtered before this point, so they never fabricate a shoreline.
  function submerged(points, level) {
    const out = [];
    points.forEach((b, i) => {
      const a = points[(i + points.length - 1) % points.length];
      if ((a.h < level) !== (b.h < level)) out.push(crossing(a, b, level));
      if (b.h < level) out.push(b);
    });
    return out;
  }
  function contour(points, level) {
    const out = [];
    points.forEach((b, i) => {
      const a = points[(i + 2) % 3];
      if ((a.h < level) !== (b.h < level)) out.push(crossing(a, b, level));
    });
    return out.length === 2 ? out : [];
  }
  function draw(ctx, tiles, sx, sy, layers, width, height) {
    function polygon(ps, fill) {
      if (ps.length < 3) return;
      ctx.beginPath(); ps.forEach((p, i) => i ? ctx.lineTo(sx(p.x), sy(p.z)) : ctx.moveTo(sx(p.x), sy(p.z)));
      ctx.closePath(); ctx.fillStyle = fill; ctx.fill();
    }
    function line(ps, colour) {
      if (ps.length !== 2) return;
      ctx.beginPath(); ctx.moveTo(sx(ps[0].x), sy(ps[0].z)); ctx.lineTo(sx(ps[1].x), sy(ps[1].z));
      ctx.strokeStyle = colour; ctx.stroke();
    }
    ctx.save(); ctx.lineWidth = 1;
    for (const tile of tiles.values()) {
      if (sx((tile.x + 1) * 16) < 0 || sx(tile.x * 16) > width || sy(tile.z * 16) < 0 || sy((tile.z + 1) * 16) > height) continue;
      for (let z = 0; z < 4; z++) for (let x = 0; x < 4; x++) {
        const i = z * 5 + x, ids = [i, i + 1, i + 6, i + 5];
        if (!ids.every((j) => Number.isFinite(tile.heights[j]))) continue;
        const ps = ids.map((j) => ({ x: tile.x * 16 + (j % 5) * 4, z: tile.z * 16 + Math.floor(j / 5) * 4, h: tile.heights[j] }));
        const biome = colours[tile.biomes[i]] || "#414847";
        if (layers.terrain) polygon(ps, biome);
        const allWet = ps.every((p) => p.h < tile.waterLevel);
        if (layers.terrain && allWet) polygon(ps, tile.waterLevel - ps.reduce((a, p) => a + p.h, 0) / 4 > 2 ? "#1d414f" : "#2a5360");
        for (const tri of [[ps[0], ps[1], ps[2]], [ps[0], ps[2], ps[3]]]) {
          const hs = tri.map((p) => p.h), low = Math.min(...hs), high = Math.max(...hs);
          if (layers.terrain) {
            const depth = tile.waterLevel - hs.reduce((a, b) => a + b, 0) / 3;
            if (!allWet) polygon(submerged(tri, tile.waterLevel), depth > 2 ? "#1d414f" : "#2a5360");
            line(contour(tri, tile.waterLevel), "#85aaa4");
          }
          if (layers.elevation && high - low < 100) {
            for (let level = Math.ceil(low / 5) * 5; level <= high; level += 5)
              line(contour(tri, level), "#c1cdb12e");
          }
        }
      }
    }
    ctx.restore();
  }
  return { key, sample, contour, submerged, draw, colours };
})();
if (typeof module !== "undefined") module.exports = RenkiTerrain;
