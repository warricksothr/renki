import assert from "node:assert/strict";
import { expect } from "@playwright/test";

export const viewports = {
  desktop: { width: 1440, height: 900 },
  mobile: { width: 390, height: 844 },
};
const snapshot = (page) =>
  page.evaluate(() => {
    const s = window.renkiSnapshots.map({ maxWidth: 640, quality: 0.8 });
    return { metadata: s.metadata, data: s.data };
  });
const camera = async (page, phase, label = phase) => {
  await page.evaluate((phase) => window.renkiPreview.setCamera(phase), phase);
  await expect(page.locator(".camera-badge")).toHaveText(label);
  if (["image", "stale", "temporary"].includes(phase))
    await expect
      .poll(() =>
        page
          .locator("#camera-image")
          .evaluate((i) => i.complete && i.naturalWidth),
      )
      .toBe(960);
  else await expect(page.locator("#camera-image")).toHaveCount(0);
};
export const scenarios = {
  "video-standby": async ({ page, check }) => {
    await check(
      "No normal JPEG polling; original slate shown without configured video",
      async () => {
        await expect(page.locator(".standby-slate")).toBeVisible();
        const fits = await page
          .locator(".video-stage")
          .evaluate(
            (stage) =>
              stage.getBoundingClientRect().width <=
              stage.closest("#camera").clientWidth + 1,
          );
        assert.ok(
          fits,
          "Standby stage fits the card at narrow widths; no clipped right edge",
        );
        await expect(page.locator(".slate-reason")).toHaveText(
          "Video is not configured",
        );
        assert.equal(
          await page.evaluate(
            () =>
              renkiPreview.requests.filter((r) =>
                r.path.startsWith("/api/camera"),
              ).length,
          ),
          0,
        );
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
  },
  "video-no-source": async ({ page, check }) => {
    await page.evaluate(() => renkiPreview.setVideo("no-source"));
    await check(
      "No source keeps standby covering every game image",
      async () => {
        await expect(
          page.locator('[data-video-state="connecting"]'),
        ).toBeVisible();
        await expect(page.locator(".slate-message strong")).toHaveText(
          "Please stand by",
        );
        await expect(page.locator(".video-stage video")).toBeHidden();
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
  },
  "video-connected": async ({ page, check }) => {
    await page.evaluate(() => renkiPreview.setVideo("gameplay"));
    await check(
      "Local synthetic frames decode; transport alone does not bypass health",
      async () => {
        await expect(
          page.locator('[data-video-state="gameplay"]'),
        ).toBeVisible();
        assert.ok(
          await page
            .locator(".video-stage video")
            .evaluate((v) => v.getVideoPlaybackQuality().totalVideoFrames > 1),
        );
        await expect(page.locator(".standby-slate")).toHaveCount(0);
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
  },
  "video-reconnect": async ({ page, check, shot }) => {
    await page.evaluate(() => renkiPreview.setVideo("gameplay"));
    await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible();
    await page.locator("#camera").scrollIntoViewIfNeeded();
    await shot("connected");
    await page.evaluate(() => renkiPreview.setVideo("stalled"));
    await check(
      "Stalled video hides the frozen picture before reconnection",
      async () => {
        await expect(
          page.locator('[data-video-state="reconnecting"]'),
        ).toBeVisible({ timeout: 7000 });
        await expect(page.locator(".video-stage video")).toBeHidden();
      },
    );
    await shot("stalled");
    await page.evaluate(() => renkiPreview.setVideo("gameplay"));
    await check("Later decoded frames recover the picture", async () => {
      await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible();
    });
  },
  "video-scope": async ({ page, check, shot }) => {
    await page.evaluate(() => renkiPreview.setVideo("gameplay"));
    await expect(page.locator('[data-video-state="gameplay"]')).toBeVisible();
    await check(
      "Logout covers game immediately and preserves the media session",
      async () => {
        await page.evaluate(() => {
          renkiPreview.beforeVideoLogouts = renkiPreview.requests.filter(
            (r) => r.method === "DELETE",
          ).length;
          renkiPreview.stream.emit("status", {
            ...renkiPreview.status,
            connected: false,
            phase: "menu",
          });
        });
        await expect(page.locator(".slate-reason")).toHaveText(
          "Tas is resting",
        );
        assert.equal(
          await page.evaluate(
            () =>
              renkiPreview.requests.filter((r) => r.method === "DELETE")
                .length - renkiPreview.beforeVideoLogouts,
          ),
          0,
        );
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
    await shot("resting");
  },

  overview: async ({ page, check }) => {
    await check("Recorded JPEG really decodes at 960×540", async () => {
      await expect(page.locator("#camera-image")).toHaveJSProperty(
        "naturalWidth",
        960,
      );
      await expect(page.locator("#camera-image")).toHaveJSProperty(
        "naturalHeight",
        540,
      );
    });
    await check(
      "Shared renderer contains all 134 recorded terrain tiles",
      async () => {
        assert.equal((await snapshot(page)).data.tiles.length, 134);
      },
    );
    await check(
      "Camera interval rejects zero and values above 300",
      async () => {
        const input = page.locator("#camera-interval");
        for (const value of ["0", "301"]) {
          await input.fill(value);
          assert.equal(await input.evaluate((i) => i.validity.valid), false);
        }
        await input.fill("5");
      },
    );
  },
  "shared-regression": async ({ page, check }) => {
    await check("PR102 fixture behavior checks", async () => {
      const result = await page.evaluate(() => window.renkiPreview.check());
      assert.ok(result.passed >= 35);
    });
  },
  "map-layers-export": async ({ page, check, output }) => {
    await check(
      "Every layer toggles aria state and exported renderer metadata",
      async () => {
        for (const layer of await page.locator("[data-layer]").all()) {
          const key = await layer.getAttribute("data-layer");
          const before = (await snapshot(page)).metadata.layers[key];
          await layer.click();
          assert.equal((await snapshot(page)).metadata.layers[key], !before);
          await expect(layer).toHaveAttribute("aria-pressed", String(!before));
          await layer.click();
        }
      },
    );
    await check(
      "Save map downloads a decoded JPEG and matching JSON from the displayed viewport",
      async () => {
        const downloads = [];
        const onDownload = (d) => downloads.push(d);
        page.on("download", onDownload);
        try {
          await page.locator("#map-export").click();
          await expect.poll(() => downloads.length).toBe(2);
          const { readFile } = await import("node:fs/promises");
          for (const d of downloads)
            await d.saveAs(output(d.suggestedFilename()));
          const json = downloads.find((d) =>
            d.suggestedFilename().endsWith(".json"),
          );
          const data = JSON.parse(
            await readFile(output(json.suggestedFilename()), "utf8"),
          );
          const jpeg = downloads.find((d) =>
            d.suggestedFilename().endsWith(".jpg"),
          );
          assert.deepEqual(
            await readFile(output(jpeg.suggestedFilename())),
            Buffer.from(data.image.split(",")[1], "base64"),
          );
          assert.deepEqual(
            data.metadata.centre,
            (await snapshot(page)).metadata.centre,
          );
          assert.equal(data.data.tiles.length, 134);
          assert.deepEqual(
            data.metadata.layers,
            (await snapshot(page)).metadata.layers,
          );
          assert.ok(
            await page.evaluate(async (image) => {
              const i = new Image();
              i.src = image;
              await i.decode();
              return i.naturalWidth > 0;
            }, data.image),
          );
        } finally {
          page.off("download", onDownload);
        }
      },
    );
    await page.locator("#map-pane").scrollIntoViewIfNeeded();
  },
  "offline-fog": async ({ page, check }) => {
    await page.evaluate(() =>
      window.renkiPreview.stream.emit("status", {
        connected: false,
        phase: "menu",
        connection: 1,
        owners: ["Nimah"],
      }),
    );
    await check(
      "Offline clears camera, actors, inventory and safety actions; retains fogged atlas",
      async () => {
        await expect(page.locator("#camera-image")).toHaveCount(0);
        await expect
          .poll(async () => (await snapshot(page)).metadata.saved)
          .toBe(true);
        const s = await snapshot(page);
        assert.equal(s.metadata.online, false);
        assert.equal(s.data.bot, null);
        assert.equal(s.data.characters.length, 0);
        assert.ok(s.data.tiles.some((t) => t.observedAt?.length));
        for (const b of await page.locator(".act").all())
          await expect(b).toBeDisabled();
        await expect(page.locator("#inv-summary")).toContainText("offline");
      },
    );
    await page.locator("#map-pane").scrollIntoViewIfNeeded();
  },
  ...Object.fromEntries(
    ["disabled", "pending", "failed", "broken", "stale"].map((phase) => [
      `camera-${phase}`,
      async ({ page, check }) => {
        await check(
          `Camera ${phase} presentation and image clearing/decoding`,
          () => camera(page, phase, phase === "broken" ? "failed" : phase),
        );
        await page.locator("#camera").scrollIntoViewIfNeeded();
      },
    ]),
  ),
  "camera-no-image": async ({ page, check }) => {
    await check(
      "Missing recorded image bytes presents disabled capture without a scene",
      async () => {
        await page.evaluate(() => {
          window.renkiPreview.jpeg = "";
          window.renkiPreview.setCamera("image");
        });
        await expect(page.locator(".camera-badge")).toHaveText("disabled");
        await expect(page.locator("#camera-image")).toHaveCount(0);
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
  },
  "camera-temporary-expiry": async ({ page, check, shot }) => {
    await check(
      "Temporary capture shows effective interval, base editor and expiry",
      async () => {
        await camera(page, "temporary", "latest frame");
        await expect(page.locator(".camera-override")).toContainText(
          "every 1s · base every 5s",
        );
        await expect(page.locator(".camera-override")).toContainText(
          "Ends in 30s",
        );
        await expect(page.locator("#camera-interval")).toHaveValue("5");
      },
    );
    await page.locator("#camera").scrollIntoViewIfNeeded();
    await shot("temporary");
    await check(
      "Simulated server expiry restores base interval without a settings write",
      async () => {
        await page.evaluate(() => window.renkiFixtureAdvance(31000));
        await expect(page.locator(".camera-override")).toHaveCount(0);
        const config = await page.evaluate(async () =>
          (await fetch("/api/camera")).json(),
        );
        assert.equal(config.config.intervalS, 5);
        assert.equal(config.override, null);
        assert.equal(
          await page.evaluate(() =>
            window.renkiPreview.requests.some((r) => r.method !== "GET"),
          ),
          false,
        );
      },
    );
  },
  "timeline-history": async ({ page, check, shot }) => {
    await check("Failure explanation and collapsed raw details", async () => {
      await expect(page.locator("#feed .event-copy").first()).toContainText(
        "path blocked",
      );
      assert.equal(
        await page
          .locator("#feed .event-details")
          .first()
          .evaluate((e) => e.open),
        false,
      );
      await page.locator("#feed .task-link").first().click();
      await expect(page.locator("#task-view")).toBeVisible();
      await expect(page.locator("#task-title")).toContainText("failed");
    });
    await shot("task");
    await page.keyboard.press("Escape");
    await check("History paging and trace filters stay usable", async () => {
      await page.locator(".timeline-options summary").click();
      await page.locator("#day").selectOption("2026-10-30");
      await page.locator("#history .earlier").click();
      await expect(page.locator("#history .earlier")).toHaveCount(0);
      await page.locator("#day").selectOption("");
      await page.locator("input[data-cat=trace]").check();
      await expect(page.locator("#feed .trace")).not.toHaveClass(/hidden/);
    });
    await page.locator("#feed-pane").scrollIntoViewIfNeeded();
  },
  "goals-inventory": async ({ page, check }) => {
    await check(
      "Recorded goal recovery, history and resume control",
      async () => {
        await expect(page.locator("#goal-list")).toContainText("needs plan");
        await expect(page.locator("#goal-list")).toContainText(
          "invalid_target",
        );
        await page
          .locator("#goal-list")
          .getByRole("button", { name: "Resume", exact: true })
          .first()
          .click();
        await expect(page.locator("#goal-result")).toContainText(
          "Read-only preview",
        );
      },
    );
    await check(
      "Inventory keyboard descriptions, stack and equipped tool",
      async () => {
        await expect(page.locator("#inv-grid .slot")).toHaveCount(32);
        const slot = page.locator('#inv-grid [tabindex="0"]').first();
        await slot.focus();
        await expect(slot).toBeFocused();
        await expect(slot).toHaveAttribute("title", /Bronze|bronze/);
      },
    );
    await page.locator("#supplies").scrollIntoViewIfNeeded();
  },
  "chest-hover": async ({ page, check, shot }) => {
    await check(
      "Real renderer hover distinguishes known, empty and unknown chest contents",
      async () => {
        // Coordinate conversion comes from the renderer's public viewport export;
        // hover remains a real pointer action, not a second rendering path.
        for (let i = 0; i < 8; i++) await page.locator("#zoom-in").click();
        const s = await snapshot(page);
        const structures = s.data.structures.filter(
          (c) => c.kind === "container",
        );
        const containers = s.data.containers;
        for (const [name, c, expected] of [
          [
            "known",
            structures.find((c) =>
              containers.find((x) => x.entity === c.id && x.items?.length),
            ),
            /Last opened/,
          ],
          [
            "empty",
            structures.find((c) =>
              containers.find((x) => x.entity === c.id && !x.items?.length),
            ),
            /Empty when last opened/,
          ],
          [
            "unknown",
            structures.find((c) => !containers.some((x) => x.entity === c.id)),
            /Contents unknown/,
          ],
        ]) {
          assert.ok(c, `${name} recorded chest exists`);
          await page.locator("#map").scrollIntoViewIfNeeded();
          const box = await page.locator("#map").boundingBox();
          const scale = await page.evaluate(() => {
            const a = window.renkiSnapshots.map({ maxWidth: 1280 });
            return (
              (a.metadata.width /
                document.querySelector("#map").getBoundingClientRect().width) *
              a.metadata.metresPerPixel
            );
          });
          const x = box.width / 2 + (c.pos[0] - s.metadata.centre[0]) / scale;
          const y = box.height / 2 - (c.pos[2] - s.metadata.centre[1]) / scale;
          await page.mouse.move(box.x + x, box.y + y);
          await expect(page.locator("#map-tip")).toContainText(expected);
          await shot(name);
        }
      },
    );
  },
  "controls-keyboard": async ({ page, check, shot }) => {
    await check(
      "Real keyboard pans/zooms/recentres the shared map",
      async () => {
        const before = await snapshot(page);
        await page.locator("#map").focus();
        await page.keyboard.press("ArrowRight");
        assert.ok(
          (await snapshot(page)).metadata.centre[0] > before.metadata.centre[0],
        );
        await expect(page.locator("#follow-bot")).not.toBeChecked();
        await page.keyboard.press("+");
        assert.ok(
          (await snapshot(page)).metadata.metresPerPixel <
            before.metadata.metresPerPixel,
        );
        await page.keyboard.press("c");
        await expect(page.locator("#follow-bot")).toBeChecked();
      },
    );
    await check(
      "Waypoint editor submits scoped data with action protection to refusing fixture",
      async () => {
        await page.locator("#place-new").click();
        await expect(page.locator("#place-radius")).toHaveValue("0");
        await page.locator("#place-name").fill("fixture camp");
        await page
          .locator("#place-form")
          .getByRole("button", { name: "Save", exact: true })
          .click();
        await expect(page.locator("#place-result")).toContainText(
          "Read-only preview",
        );
        const r = await page.evaluate(() =>
          window.renkiPreview.requests.findLast(
            (r) => r.path === "/api/waypoints",
          ),
        );
        assert.equal(r.headers["X-Renki-Act"], "1");
        assert.equal(JSON.parse(r.body).waypoint.name, "fixture camp");
      },
    );
    await shot("waypoint");
    await check(
      "Player profile and settings errors remain readable and preserve form focus",
      async () => {
        await page.locator("#presence-controls summary").click();
        await page.locator("#profile-name").fill("Fixture guest");
        await page.locator("#profile-role").selectOption("guest");
        await page.locator("#profile-form").getByRole("button").click();
        await expect(page.locator("#presence-result")).toContainText(
          "Read-only preview",
        );
        await page.locator("#settings summary").click();
        const button = page
          .locator("#settings-table tr:last-child button")
          .last();
        await button.focus();
        await expect(button).toBeFocused();
        await expect(button).toBeInViewport();
        await button.press("Enter");
        await expect(page.locator("#settings-result")).toContainText(
          "Read-only preview",
        );
      },
    );
    await page.locator("#preferences").scrollIntoViewIfNeeded();
  },
};
export const examples = [
  "overview",
  "map-layers-export",
  "offline-fog",
  "camera-stale",
  "camera-temporary-expiry",
  "timeline-history",
  "goals-inventory",
  "chest-hover",
  "controls-keyboard",
];
