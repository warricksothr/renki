import { render } from "preact";
import { useStatus, online, humanize } from "./status.js";
import {
  MapWorkspace,
  Supplies,
  Places,
  Preferences,
  Timeline,
} from "./workspace.jsx";
import { Camera } from "./camera.jsx";
import { startController } from "./controller.jsx";

function Food() {
  const s = useStatus();
  const food = online(s) ? s.food?.active : null;
  return (
    <>
      <strong>{food ? `${food.length} / 3 meals` : "Not observed"}</strong>
      <span class="muted">
        {food?.length
          ? food
              .map(
                (f) =>
                  `${humanize(f.prefab)} · ${Math.ceil(f.remainingS / 60)}m`,
              )
              .join(" / ")
          : online(s)
            ? "Food keeps health and stamina up"
            : "Waiting for the bot"}
      </span>
    </>
  );
}
function Objective() {
  const s = useStatus();
  const goal = s?.goals?.find(
    (g) => !["succeeded", "cancelled"].includes(g.status),
  );
  return (
    <span class="muted">
      {goal
        ? `Persistent goal · ${humanize(goal.status)} · ${goal.reason || "Objective saved"}`
        : online(s)
          ? s.task
            ? "Direct order in progress"
            : "Ready for an order"
          : "No live task observation"}
    </span>
  );
}
function App() {
  return (
    <>
      <a href="#workspace" class="skip-link">
        Skip to workspace
      </a>
      <header class="topbar">
        <a class="brand" href="#workspace">
          <span class="brand-mark">r</span>
          <span>
            renki<small>CONTROL ROOM</small>
          </span>
        </a>
        <div class="identity">
          <span id="link" class="chip off">
            Connecting…
          </span>
          <span id="who" />
          <span id="phase" class="muted" />
        </div>
        <div class="acts" aria-label="Bot safety controls">
          <button class="act stop" data-act="stop" disabled>
            ■ Stop
          </button>
          <button class="act" data-act="home" disabled>
            Go home
          </button>
          <button class="act" data-act="logout" disabled>
            Log bot out
          </button>
        </div>
        <form method="post" action="/logout">
          <button class="session-out" title="End this browser's panel session">
            Sign out
          </button>
        </form>
      </header>
      <div
        class="action-feedback"
        id="act-result"
        role="status"
        aria-live="polite"
      />
      <main id="workspace" tabIndex="-1">
        <div class="workspace-title">
          <div>
            <span class="eyebrow">FIELD OPERATIONS</span>
            <h1>Field overview</h1>
          </div>
          <nav aria-label="Workspace sections">
            <a href="#map-pane">Map</a>
            <a href="#camera">Camera</a>
            <a href="#navigation">Places</a>
            <a href="#supplies">Kit &amp; goals</a>
            <a href="#feed-pane">Activity</a>
            <a href="#preferences">Preferences</a>
          </nav>
        </div>
        <div class="overview" aria-label="Bot overview">
          <div class="metric">
            <span class="eyebrow">VITALS</span>
            <strong id="hp">Health unknown</strong>
            <span id="stamina" class="muted" />
          </div>
          <div class="metric">
            <span class="eyebrow">FOOD</span>
            <Food />
          </div>
          <div class="metric objective">
            <span class="eyebrow">CURRENT WORK</span>
            <strong id="task">Waiting for status</strong>
            <Objective />
          </div>
          <div class="metric">
            <span class="eyebrow">SAFETY &amp; RETURN</span>
            <strong id="home">Home unknown</strong>
            <span id="combat" />
            <span id="load" class="chip" />
          </div>
        </div>
        <div class="visual-workspace">
          <MapWorkspace />
          <div class="observation-column">
            <Camera />
            <Timeline />
          </div>
        </div>
        <div class="operations">
          <Supplies />
          <Places />
          <Preferences />
        </div>
        <footer>
          Renki · Observations guide decisions. Task receipts confirm outcomes.
        </footer>
      </main>
    </>
  );
}
render(<App />, document.getElementById("app"));
startController();
