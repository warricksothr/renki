import { build } from "esbuild";
import { readFile, writeFile } from "node:fs/promises";
const result = await build({
  entryPoints: ["src/main.jsx"],
  bundle: true,
  outfile: "../internal/panel/static/app.js",
  format: "iife",
  target: ["es2020"],
  jsx: "automatic",
  jsxImportSource: "preact",
  minify: true,
  legalComments: "eof",
  write: false,
});
const outputs = [
  ["../internal/panel/static/app.js", result.outputFiles[0].contents],
  ["../internal/panel/static/style.css", await readFile("src/style.css")],
];
for (const [path, contents] of outputs) {
  if (process.argv.includes("--check")) {
    if (!Buffer.from(contents).equals(await readFile(path)))
      throw Error(`${path} is stale; run just panel build`);
  } else await writeFile(path, contents);
}
console.log(
  `Panel: ${outputs[0][1].length} bytes JS, ${outputs[1][1].length} bytes CSS`,
);
