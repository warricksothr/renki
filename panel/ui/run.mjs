import { chromium, expect } from "@playwright/test";
import { createServer } from "node:http";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { resolve, join, basename } from "node:path";
import { createRequire } from "node:module";
import { scenarios, viewports, examples } from "./scenarios.mjs";

const require = createRequire(import.meta.url);
const args = process.argv.slice(2);
const options = {
  viewport: "all",
  scenario: "all",
  output: "../.checks/panel-ui",
  port: "18792",
};
for (let i = 0; i < args.length; i++) {
  const key = args[i].replace(/^--/, "");
  if (key === "examples") options.examples = true;
  else if (Object.hasOwn(options, key) && args[i + 1]) options[key] = args[++i];
  else
    throw Error(
      `Unknown/incomplete option ${args[i]}; use --scenario NAME --viewport desktop|mobile|all --output DIR --port PORT --examples`,
    );
}
if (options.examples && options.output === "../.checks/panel-ui")
  options.output = "../.checks/panel-examples";
const selected =
  options.scenario === "all"
    ? options.examples
      ? examples
      : Object.keys(scenarios)
    : options.scenario.split(",");
const sizes =
  options.viewport === "all" ? Object.keys(viewports) : [options.viewport];
for (const name of selected)
  if (!scenarios[name])
    throw Error(
      `Unknown scenario ${name}; choose ${Object.keys(scenarios).join(", ")}`,
    );
for (const name of sizes)
  if (!viewports[name])
    throw Error(`Unknown viewport ${name}; choose desktop, mobile, all`);
if (!/^\d+$/.test(options.port) || Number(options.port) > 65535)
  throw Error("Port must be 0–65535; zero selects an available local port");
const out = resolve(options.output);
await mkdir(out, { recursive: true });
const result = {
  schemaVersion: 1,
  status: "running",
  fixture: "recorded PR102 observations + simulated events/timing; NOT LIVE",
  metadata: {},
  results: [],
  visualReview: "unreviewed; use visual-review.json and inspect screenshots",
  fatal: null,
};
let browser, server;
const hash = async (p) =>
  createHash("sha256")
    .update(await readFile(p))
    .digest("hex");
const save = async () => {
  const compact = {
    ...result,
    results: result.results.map((r) => ({
      ...r,
      failures: r.failures.map((f) => ({ message: f.message.slice(0, 900) })),
      errorCounts: Object.fromEntries(
        Object.entries(r.errors).map(([k, v]) => [k, v.length]),
      ),
      errors: Object.fromEntries(
        Object.entries(r.errors).map(([k, v]) => [k, v.slice(0, 2)]),
      ),
    })),
  };
  await writeFile(
    join(out, "result.json"),
    JSON.stringify(compact, null, 2) + "\n",
  );
};
const bounded = async (work, ms) => {
  let timer;
  try {
    return await Promise.race([
      work,
      new Promise((_, reject) => {
        timer = setTimeout(
          () => reject(Error(`Scenario exceeded ${ms}ms; inspect diagnostics`)),
          ms,
        );
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
};
try {
  result.metadata = {
    commit: execFileSync("git", ["rev-parse", "HEAD"], {
      encoding: "utf8",
    }).trim(),
    dirty: !!execFileSync("git", ["status", "--porcelain"], {
      encoding: "utf8",
    }).trim(),
    node: process.version,
    platform: `${process.platform}/${process.arch}`,
    playwright: require("@playwright/test/package.json").version,
    chromiumExecutable: basename(resolve(chromium.executablePath(), "../..")),
    locale: "en-US",
    timezone: "UTC",
    deviceScaleFactor: 1,
    clock: "2026-10-30T19:00:00Z",
    seed: 102,
    font: "Inter Latin 400 (locked @fontsource/inter)",
    hashes: Object.fromEntries(
      await Promise.all(
        [
          "fixtures/status.json",
          "fixtures/mapped.json",
          "fixtures/atlas.json",
          "fixtures/camera.jpg",
          "../internal/panel/static/app.js",
          "../internal/panel/static/terrain.js",
          "../internal/panel/static/style.css",
          "node_modules/@fontsource/inter/files/inter-latin-400-normal.woff2",
        ].map(async (p) => [p, await hash(p)]),
      ),
    ),
  };
  // Reuse the existing preview builder and exact built production assets.
  execFileSync(
    "just",
    [
      "--no-deps",
      "controller",
      "preview-map",
      "--suite",
      "--output",
      join(out, "fixture.html"),
    ],
    { cwd: "..", stdio: "pipe" },
  );
  const html = await readFile(join(out, "fixture.html"));
  server = createServer(async (req, res) => {
    if (req.url === "/" && req.method === "GET") {
      res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
      res.end(html);
    } else if (
      req.method === "GET" &&
      /^\/artifacts\/[a-zA-Z0-9_.-]+\.(png|html)$/.test(req.url)
    ) {
      try {
        const data = await readFile(join(out, basename(req.url)));
        res.writeHead(200, {
          "Content-Type": req.url.endsWith(".png")
            ? "image/png"
            : "text/html; charset=utf-8",
        });
        res.end(data);
      } catch {
        res.writeHead(404);
        res.end();
      }
    } else {
      res.writeHead(405);
      res.end("Read-only fixture; unexpected request refused");
    }
  });
  await new Promise((yes, no) => {
    server.once("error", no);
    server.listen(Number(options.port), "127.0.0.1", yes);
  });
  const origin = `http://127.0.0.1:${server.address().port}`;
  try {
    browser = await chromium.launch({ headless: true });
  } catch (e) {
    throw Error(
      `Locked Chromium could not launch. Run just panel ui-install; install the OS libraries named by Playwright if needed. Browser tests did NOT pass or skip.\n${e.message}`,
    );
  }
  result.metadata.browser = browser.version();
  for (const viewport of sizes)
    for (const scenario of selected) {
      const entry = {
        scenario,
        viewport,
        status: "running",
        checks: [],
        failures: [],
        screenshots: [],
        errors: { runtime: [], network: [], console: [] },
        diagnostics: `${viewport}-${scenario}.diagnostics.json`,
      };
      result.results.push(entry);
      const context = await browser.newContext({
        viewport: viewports[viewport],
        deviceScaleFactor: 1,
        locale: "en-US",
        timezoneId: "UTC",
        colorScheme: "dark",
        reducedMotion: "reduce",
        serviceWorkers: "block",
        acceptDownloads: true,
      });
      const page = await context.newPage();
      page.setDefaultTimeout(7000);
      const logs = [];
      page.on("pageerror", (e) =>
        entry.errors.runtime.push(e.stack || e.message),
      );
      page.on("console", (m) => {
        logs.push({ type: m.type(), text: m.text() });
        if (m.type() === "error") entry.errors.console.push(m.text());
      });
      page.on("requestfailed", (r) =>
        entry.errors.network.push({
          url: r.url(),
          error: r.failure()?.errorText,
        }),
      );
      page.on("response", (r) => {
        if (r.status() >= 400)
          entry.errors.network.push({ url: r.url(), status: r.status() });
      });
      await context.route("**/*", (route) => {
        const request = route.request();
        if (request.url() === origin + "/" && request.method() === "GET")
          return route.continue();
        entry.errors.network.push({
          url: request.url(),
          error: "Unexpected real network request blocked",
        });
        return route.abort("blockedbyclient");
      });
      const shot = async (suffix = "result") => {
        const name = `${viewport}-${scenario}-${suffix}.png`;
        await page.screenshot({
          path: join(out, name),
          animations: "disabled",
          caret: "hide",
        });
        entry.screenshots.push(name);
      };
      try {
        await page.goto(origin + "/", { waitUntil: "load" });
        await page.evaluate(async () => {
          await window.renkiPreviewReady;
          await document.fonts.ready;
        });
        if (!scenario.startsWith("video-")) {
          await page.locator(".video-diagnostics > summary").click();
          await expect(page.locator(".camera-badge")).toHaveText(
            "latest frame",
          );
        }
        await expect
          .poll(() =>
            page.evaluate(
              () =>
                !!window.renkiSnapshots &&
                window.renkiSnapshots.map().data.tiles.length,
            ),
          )
          .toBe(134);
        await bounded(
          scenarios[scenario]({
            page,
            shot,
            output: (name) => join(out, `${viewport}-${scenario}-${name}`),
            check: async (name, fn) => {
              await fn();
              entry.checks.push(name);
            },
          }),
          60000,
        );
        {
          const width = await page.evaluate(() => ({
            page: document.body.scrollWidth,
            viewport: innerWidth,
          }));
          if (width.page > width.viewport + 1)
            throw Error(
              `Horizontal overflow ${width.page} > ${width.viewport}`,
            );
          entry.checks.push("No horizontal document overflow");
        }
        if (scenario === "overview")
          await page.evaluate(() => window.scrollTo(0, 0));
        await shot();
      } catch (e) {
        entry.failures.push({ message: e.message, stack: e.stack });
        try {
          await shot("failure");
        } catch (e) {
          entry.failures.push({ message: `Failure screenshot: ${e.message}` });
        }
      } finally {
        let fixture = {};
        try {
          fixture = await page.evaluate(() => ({
            requests: window.renkiPreview?.requests,
            errors: window.renkiPreview?.errors,
            camera: document.querySelector(".camera-badge")?.textContent,
            viewport: { width: innerWidth, height: innerHeight },
            fonts: document.fonts.status,
          }));
        } catch {
          /* Navigation failure is already retained above. */
        }
        if (fixture.errors?.length)
          entry.errors.runtime.push(...fixture.errors);
        entry.status =
          entry.failures.length ||
          Object.values(entry.errors).some((v) => v.length)
            ? "failed"
            : "passed";
        await writeFile(
          join(out, entry.diagnostics),
          JSON.stringify({ ...entry, logs, fixture }, null, 2) + "\n",
        );
        await context.close();
        await save();
        console.log(
          `${entry.status.toUpperCase()} ${viewport}/${scenario}: ${entry.checks.length} checks; ${entry.screenshots.join(", ")}`,
        );
        for (const failure of entry.failures.slice(0, 2))
          console.log(failure.message.slice(0, 900));
        for (const [kind, errors] of Object.entries(entry.errors))
          if (errors.length)
            console.log(
              `${kind}: ${JSON.stringify(errors.slice(0, 2)).slice(0, 600)}; full: ${entry.diagnostics}`,
            );
      }
    }
  const shots = result.results.flatMap((r) =>
    r.screenshots.map((path) => ({
      path,
      scenario: r.scenario,
      viewport: r.viewport,
      status: r.status,
    })),
  );
  await writeFile(
    join(out, "visual-review.json"),
    JSON.stringify(
      {
        status: "unreviewed",
        reviewer: null,
        environment: result.metadata,
        checklist: [
          "Hierarchy, labels and fixture-not-live notice are clear",
          "No clipped controls, overlapping text or unreadable tooltip at either size",
          "Map layers/fog and camera states communicate the right scope/freshness",
          "Focus, scroll, error messages, timeline and mobile safety controls remain usable",
          "Examples show the intended feature; compare to approved design and explain concerns",
        ],
        screenshots: shots.map((s) => ({
          ...s,
          visualConcerns: [],
          decision: "unreviewed",
        })),
        note: "Semantic success, model opinions and pixel similarity do not establish visual correctness. Record specific concerns and evidence paths. No automatic baseline approval.",
      },
      null,
      2,
    ) + "\n",
  );
  const sheet = `<!doctype html><html lang="en"><meta charset="utf-8"><title>Renki fixture examples</title><style>body{background:#172019;color:#eee;font:16px sans-serif;margin:24px}main{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}figure{margin:0}img{width:100%}a{color:#bfe0b2}small{display:block}</style><h1>FIXTURE · NOT LIVE</h1><p>Recorded PR102 map/JPEG + simulated events/timing. Visual review required. ${result.metadata.browser} / Playwright ${result.metadata.playwright}.</p><main>${shots.map((s) => `<figure><a href="${s.path}"><img src="${s.path}" alt="${s.viewport} ${s.scenario}"></a><figcaption>${s.viewport} / ${s.scenario}<small>${s.path} · ${s.status}</small></figcaption></figure>`).join("")}</main></html>`;
  await writeFile(join(out, "contact-sheet.html"), sheet);
  const sheetPage = await browser.newPage({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 1,
  });
  await sheetPage.goto(origin + "/artifacts/contact-sheet.html");
  await sheetPage.evaluate(() =>
    Promise.all([...document.images].map((i) => i.decode())),
  );
  await sheetPage.screenshot({
    path: join(out, "contact-sheet.png"),
    fullPage: true,
  });
  await sheetPage.close();
  result.status =
    result.results.length === selected.length * sizes.length &&
    result.results.length > 0 &&
    result.results.every((r) => r.status === "passed")
      ? "passed"
      : "failed";
} catch (e) {
  result.status = "failed";
  result.fatal = e.stack || e.message;
  console.error(String(e.message).slice(0, 1800));
} finally {
  await browser?.close();
  if (server?.listening) await new Promise((yes) => server.close(yes));
  await save();
}
console.log(
  `${result.status.toUpperCase()}: ${result.results.filter((r) => r.status === "passed").length}/${result.results.length} scenarios; ${join(out, "result.json")}`,
);
if (result.status !== "passed") process.exitCode = 1;
