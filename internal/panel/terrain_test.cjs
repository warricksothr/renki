const { test } = require('node:test');
const assert = require('node:assert/strict');
const { sample, key, contour, submerged } = require('./static/terrain.js');

test('negative tile coordinates and bilinear interpolation preserve height', () => {
  const t = { x: -1, z: -2, waterLevel: 30, heights: Array.from({length:25}, (_, i) => i % 5 + Math.floor(i/5)*2), biomes: Array(25).fill('Meadows') };
  assert.equal(sample(new Map([[key(-1, -2), t]]), -14, -30).height, 1.5);
  t.heights[0] = null;
  assert.equal(sample(new Map([[key(-1, -2), t]]), -14, -30), null);
  assert.equal(sample(new Map(), 0, 0), null);
});

test('shoreline clipping intersects edges rather than painting a dry triangle wet', () => {
  const tri = [{x:0,z:0,h:0}, {x:4,z:0,h:4}, {x:0,z:4,h:4}];
  const line = contour(tri, 2);
  assert.equal(line.length, 2);
  assert.deepEqual(line.map(p => [p.x,p.z]).sort(), [[0,2],[2,0]]);
  const wet = submerged(tri, 2);
  assert.equal(wet.length, 3);
  assert(wet.every(p => p.h <= 2));
  assert.equal(submerged(tri, -1).length, 0);
  assert.equal(submerged(tri, 5).length, 3);
});
