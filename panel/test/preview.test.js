import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { gunzipSync } from "node:zlib";
test("standalone preview embeds the production bundle and lossless compressed read-only fixtures", () => {
  const folder = mkdtempSync(join(tmpdir(), "renki-preview-"));
  try {
    const out = join(folder, "preview.html");
    // Test the checked-in assets without reinstalling dependencies under the
    // other Node test workers; the CI verify recipe checks those assets first.
    execFileSync(
      "just",
      ["--no-deps", "controller", "preview-map", "--output", out],
      { cwd: "..", stdio: "pipe" },
    );
    const html = readFileSync(out, "utf8");
    const packed = html.match(/atob\('([A-Za-z0-9+/=]+)'\)/)[1];
    const fixture = JSON.parse(gunzipSync(Buffer.from(packed, "base64")));
    assert.equal(fixture.mapped.tiles.length, 49);
    assert.equal(fixture.status.world, "DizzysHellheim");
    assert.deepEqual(fixture.mapped.containers[0].items[0], {
      prefab: "CookedDeerMeat",
      stack: 9,
    });
    assert.ok(
      html.includes(readFileSync("../internal/panel/static/app.js", "utf8")),
    );
    assert.ok(html.includes("Read-only preview: actions are refused"));
    assert.ok(
      html.length < 512000,
      "preview must fit the native HTML tool input limit",
    );
    assert.ok(!html.includes('src="/app.js"'));
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
});
