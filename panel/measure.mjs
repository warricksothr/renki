import { execFileSync } from "node:child_process";
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { gzipSync } from "node:zlib";
const baseline = process.argv[2] || "origin/main";
const assets = {};
for (const name of ["app.js", "style.css", "index.html"]) {
  const path = `internal/panel/static/${name}`;
  const before = execFileSync("git", ["show", `${baseline}:${path}`]);
  const after = await readFile(`../${path}`);
  assets[name] = {
    beforeBytes: before.length,
    afterBytes: after.length,
    beforeGzip: gzipSync(before, { level: 9 }).length,
    afterGzip: gzipSync(after, { level: 9 }).length,
  };
}
await mkdir("../.checks", { recursive: true });
await writeFile(
  "../.checks/panel-build-impact.json",
  JSON.stringify({ baseline, assets }, null, 2) + "\n",
);
console.log(JSON.stringify(assets));
