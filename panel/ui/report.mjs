import { readFileSync } from "node:fs";
const [path = "../.checks/panel-ui/result.json", scenario] =
  process.argv.slice(2);
const result = JSON.parse(readFileSync(path, "utf8"));
console.log(
  JSON.stringify(
    {
      status: result.status,
      fatal: result.fatal,
      metadata: result.metadata,
      results: result.results
        .filter((r) => !scenario || r.scenario === scenario)
        .map((r) => ({
          scenario: r.scenario,
          viewport: r.viewport,
          status: r.status,
          checks: r.checks.length,
          failures: r.failures.slice(0, 3),
          errors:
            r.errorCounts ||
            Object.fromEntries(
              Object.entries(r.errors).map(([k, v]) => [k, v.length]),
            ),
          screenshots: r.screenshots,
          diagnostics: r.diagnostics,
        })),
    },
    null,
    2,
  ),
);
