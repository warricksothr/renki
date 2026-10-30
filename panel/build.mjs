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
const slate = await build({
  entryPoints: ["src/slate.jsx"],
  bundle: true,
  write: false,
  format: "iife",
  jsx: "automatic",
  jsxImportSource: "preact",
  minify: true,
  target: ["es2020"],
});
const standbyCSS = await readFile("src/standby.css", "utf8");
const slateHTML = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Renki — Please stand by</title><style>html,body,#slate{margin:0;width:100%;height:100%;overflow:hidden}${standbyCSS}</style><body><main id="slate"></main><script>${slate.outputFiles[0].text}</script></body></html>`;
const outputs = [
  ["../internal/panel/static/slate.html", Buffer.from(slateHTML)],
  ["../internal/panel/static/app.js", result.outputFiles[0].contents],
  [
    "../internal/panel/static/style.css",
    Buffer.from((await readFile("src/style.css", "utf8")) + "\n" + standbyCSS),
  ],
];
for (const [path, contents] of outputs) {
  if (process.argv.includes("--check")) {
    if (!Buffer.from(contents).equals(await readFile(path)))
      throw Error(`${path} is stale; run just panel build`);
  } else await writeFile(path, contents);
}
console.log(
  `Panel: ${outputs[1][1].length} bytes JS, ${outputs[2][1].length} bytes CSS`,
);
