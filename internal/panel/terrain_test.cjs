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

test('remembered corners retain their ages when merged with complete-looking live tiles', () => {
  const { merge } = require('./static/terrain.js');
  const live = {x:0,z:0,waterLevel:30,heights:Array(25).fill(5),biomes:Array(25).fill('Meadows')};
  const saved = {...live,heights:Array(25).fill(9),observedAt:Array(25).fill('2026-10-01T00:00:00Z'),sources:Array(25).fill('bot'),stale:Array(25).fill(false)};
  saved.heights[24]=null;
  saved.observedAt[1]='2026-09-01T00:00:00Z';saved.stale[1]=true;
  const tiles=merge([live],[saved]);
  assert.equal(tiles.get(key(0,0)).heights[24],5);
  const s=sample(tiles,1,1);
  assert.equal(s.height,9);
  assert.equal(s.observedAt,'2026-09-01T00:00:00Z');
  assert.equal(s.stale,true);
  assert.deepEqual(s.sources,['bot']);
  assert.equal(live.heights[0],5);
  assert.equal(saved.heights[24],null);
});

test('offline fog dims known terrain without painting unknown cells', () => {
  const { draw } = require('./static/terrain.js');
  const tile={x:0,z:0,waterLevel:30,heights:Array(25).fill(40),biomes:Array(25).fill('Meadows')};
  tile.heights[0]=null;
  function paint(fog){
    const alpha=[];
    const ctx={save(){},restore(){},beginPath(){},lineTo(){},moveTo(){},closePath(){},stroke(){},fill(){alpha.push(this.globalAlpha)}};
    draw(ctx,new Map([[key(0,0),tile]]),x=>x,z=>32-z,{terrain:true,elevation:false},32,32,fog);
    return alpha;
  }
  const live=paint(false),offline=paint(true);
  assert.equal(live.length,15);
  assert.equal(offline.length,15);
  assert(live.every(a=>a===1));
  assert(offline.every(a=>a>0 && a<1));
});
