// Runs only in the self-contained, read-only fixture. Invoke through the native
// browser console: await renkiPreview.check(). No live server can load this file.
window.renkiPreview.check = async () => {
  const fixture = window.renkiPreview;
  const results = [];
  const assert = (condition, name) => {
    if (!condition) throw Error(name);
    results.push(name);
  };
  const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
  await wait(150);
  const initial = { ...fixture.status };
  const initialCamera = fixture.cameraState;
  fixture.temporaryExpires = Date.now() + 30000;
  const map = () => window.renkiSnapshots.map({ maxWidth: 640, quality: 0.8 });
  const canvas = document.getElementById("map");
  const initialShot = map();
  assert(
    initialShot.image.startsWith("data:image/jpeg"),
    "Shared canvas snapshot produces JPEG",
  );
  assert(
    initialShot.metadata.width <= 640 && initialShot.data.tiles.length > 0,
    "Snapshot retains viewport and measured terrain",
  );
  assert(
    document.body.scrollWidth <= innerWidth + 1,
    "No horizontal page overflow",
  );
  for (const layer of document.querySelectorAll("[data-layer]")) {
    const before = map().metadata.layers[layer.dataset.layer];
    layer.click();
    assert(
      map().metadata.layers[layer.dataset.layer] === !before,
      `${layer.dataset.layer} layer uses export renderer`,
    );
    layer.click();
  }
  canvas.focus();
  canvas.dispatchEvent(
    new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }),
  );
  const panned = map();
  assert(
    panned.metadata.centre[0] > initialShot.metadata.centre[0] &&
      !document.getElementById("follow-bot").checked,
    "Keyboard pans actual map viewport",
  );
  canvas.dispatchEvent(
    new KeyboardEvent("keydown", { key: "c", bubbles: true }),
  );
  assert(
    document.getElementById("follow-bot").checked,
    "Keyboard restores bot following",
  );
  const beforeZoom = map().metadata.metresPerPixel;
  canvas.dispatchEvent(
    new KeyboardEvent("keydown", { key: "+", bubbles: true }),
  );
  assert(
    map().metadata.metresPerPixel < beforeZoom,
    "Keyboard zoom changes snapshot scale",
  );
  document.getElementById("zoom-out").click();
  const day = document.getElementById("day");
  day.value = "2026-10-30";
  day.dispatchEvent(new Event("change"));
  await wait(50);
  assert(
    !document.getElementById("history").hidden &&
      document.querySelector("#history .earlier"),
    "Audit history and pagination remain accessible",
  );
  document.querySelector("#history .earlier").click();
  await wait(50);
  assert(
    !document.querySelector("#history .earlier"),
    "Earlier audit page loads",
  );
  day.value = "";
  day.dispatchEvent(new Event("change"));
  const task = document.querySelector("#feed .task-link");
  task.click();
  await wait(50);
  assert(
    document.getElementById("task-view").open &&
      document.getElementById("task-title").textContent.includes("failed"),
    "Task drilldown retains run-scoped history",
  );
  document.getElementById("task-close").click();
  assert(
    !document.querySelector("#feed .event-details").open,
    "Raw technical details collapsed by default",
  );
  assert(
    document
      .querySelector("#feed .event-copy")
      .textContent.includes("path blocked"),
    "Task failure reason is human-readable",
  );
  assert(
    document.querySelector("#feed .trace").classList.contains("hidden"),
    "Routine traces hidden by default",
  );
  const trace = document.querySelector("[data-cat=trace]");
  trace.click();
  assert(
    !document.querySelector("#feed .trace").classList.contains("hidden"),
    "Trace filter still available",
  );
  trace.click();
  for (const id of [
    "inventory",
    "places",
    "presence-controls",
    "voice",
    "settings",
  ]) {
    const section = document.getElementById(id);
    section.open = true;
  }
  await wait(50);
  assert(
    document.querySelectorAll("#settings-table tr").length ===
      fixture.settings.length,
    "All runtime settings reachable",
  );
  const last = document.querySelector("#settings-table tr:last-child button");
  last.scrollIntoView({ block: "center" });
  last.focus();
  assert(
    document.activeElement === last &&
      last.getBoundingClientRect().bottom <= innerHeight,
    "Long settings panel scrolls to keyboard focus",
  );
  assert(
    document.querySelectorAll("#inv-grid .slot").length === 32 &&
      document.querySelector('#inv-grid [tabindex="0"]'),
    "Inventory slots and item keyboard descriptions retained",
  );
  if (fixture.jpeg) {
    for (const phase of [
      "disabled",
      "pending",
      "failed",
      "broken",
      "stale",
      "temporary",
      "image",
    ]) {
      fixture.cameraState = phase;
      await wait(1200);
      const label = document.querySelector(".camera-badge").textContent;
      if (phase === "temporary") {
        assert(
          document
            .querySelector(".camera-override")
            ?.textContent.includes("base every 5s"),
          "Temporary capture shows effective and base intervals",
        );
        assert(
          document
            .querySelector(".camera-override")
            ?.textContent.includes("Ends in") &&
            document.getElementById("camera-interval").value === "5",
          "Temporary expiry preserves the base setting editor",
        );
      } else if (phase === "image")
        assert(
          document.getElementById("camera-image")?.naturalWidth > 0 &&
            label === "latest frame",
          "Actual JPEG loads and decodes under image CSP",
        );
      else if (phase === "stale")
        assert(label === "stale", "Old scene is explicitly stale");
      else if (phase === "broken")
        assert(
          label === "failed" && !document.getElementById("camera-image"),
          "Image decode error is visible and clears scene",
        );
      else
        assert(
          label === phase && !document.getElementById("camera-image"),
          `${phase} camera state clears scene`,
        );
    }
  }
  fixture.stream.emit("status", {
    connected: false,
    phase: "menu",
    connection: initial.connection,
    owners: initial.owners,
  });
  await wait(100);
  assert(
    !document.getElementById("camera-image"),
    "Offline clears camera immediately",
  );
  const saved = map();
  assert(
    saved.metadata.saved &&
      !saved.metadata.online &&
      saved.data.bot === null &&
      saved.data.characters.length === 0,
    "Offline fogged atlas excludes live actors",
  );
  assert(
    [...document.querySelectorAll(".act")].every((b) => b.disabled),
    "Offline safety actions disabled",
  );
  fixture.cameraState = initialCamera;
  fixture.stream.emit("status", initial);
  await wait(1200);
  assert(
    !map().metadata.saved && map().metadata.world === initial.world,
    "Rejoin restores live scope",
  );
  fixture.stream.emit("record", {
    kind: "frame",
    cat: "perception",
    frame: fixture.frames.find((f) => f.type === "inventory"),
  });
  // Mutations go only to the in-document fixture, which always refuses them.
  const response = await fetch("/api/act/stop", {
    method: "POST",
    headers: { "X-Renki-Act": "1" },
  });
  assert(response.status === 405, "Fixture rejects game actions");
  assert(fixture.errors.length === 0, "No uncaught browser errors");
  for (const id of ["presence-controls", "voice", "settings"])
    document.getElementById(id).open = false;
  window.scrollTo(0, 0);
  return {
    passed: results.length,
    checks: results,
    viewport: { width: innerWidth, height: innerHeight },
    cameraWidth: document.getElementById("camera-image")?.naturalWidth || 0,
  };
};
