// Static control boundaries keep the existing API/map adapters attached once.
// Preact owns the workspace; the adapter owns children of its named containers.
export function MapWorkspace() {
  return (
    <section id="map-pane">
      <div class="map-heading">
        <div>
          <span class="eyebrow">WORLD MAP</span>
          <strong id="map-name">Location</strong>
          <span id="map-world"></span>
        </div>
        <div class="location-chips">
          <span id="biome" class="chip"></span>
          <span id="surface" class="chip"></span>
          <span id="shelter" class="chip"></span>
        </div>
      </div>
      <div class="map-layers" aria-label="Map layers">
        <button data-layer="terrain" aria-pressed="true">
          Terrain
        </button>
        <button data-layer="elevation" aria-pressed="true">
          Elevation
        </button>
        <button data-layer="memory" aria-pressed="true">
          Remembered
        </button>
        <button data-layer="danger" aria-pressed="true">
          Danger
        </button>
        <button data-layer="resources" aria-pressed="false">
          Resources
        </button>
        <button data-layer="obstacles" aria-pressed="true">
          Log boundaries
        </button>
        <label>
          Map{" "}
          <select id="atlas-world" aria-label="Map world">
            <option value="">Follow Tas · last map offline</option>
          </select>
        </label>
        <span id="terrain-status">Waiting for terrain</span>
      </div>
      <canvas
        id="map"
        tabIndex="0"
        aria-label="Top-down map around the bot. Arrow keys pan, plus and minus zoom, C centres on bot."
      ></canvas>
      <div id="map-tip" hidden></div>
      <div class="location-details">
        <span id="height">Height unknown</span>
        <span id="depth"></span>
      </div>
      <div class="map-bar">
        <span id="pos"></span>
        <label>
          <input type="checkbox" id="follow-bot" checked /> Centre on the bot
        </label>
        <button
          id="map-export"
          title="Save the displayed map and matching data"
        >
          Save map snapshot
        </button>
        <button id="zoom-in" title="Zoom in">
          +
        </button>
        <button id="zoom-out" title="Zoom out">
          −
        </button>
      </div>
      <div class="terrain-legend">
        <span class="meadows">Meadows</span>
        <span class="forest">Black Forest</span>
        <span class="water">Water</span>
        <span class="unknown">Unknown terrain</span>
        <span class="memory">Faint = last seen</span>
        <span>Amber = log boundary and body clearance</span>
      </div>
      <details class="map-key">
        <summary>Map key</summary>
        <ul class="legend">
          <li>
            <i class="bot"></i>bot and its trail
          </li>
          <li>
            <i class="player"></i>player
          </li>
          <li>
            <i class="other"></i>creature
          </li>
          <li>
            <i class="hostile"></i>hostile (ringed if after the bot)
          </li>
          <li>
            <i class="tamed"></i>tamed
          </li>
          <li>
            <i class="chest"></i>chest
          </li>
          <li>
            <i class="station"></i>station
          </li>
          <li>
            <i class="bed"></i>bed
          </li>
          <li>
            <i class="fire"></i>fire
          </li>
          <li>
            <i class="smelter"></i>smelter
          </li>
          <li>
            <i class="portal"></i>portal
          </li>
          <li>
            <i class="door"></i>door (hollow when open)
          </li>
          <li>
            <i class="wall"></i>wall
          </li>
          <li>
            <i class="tree"></i>tree
          </li>
          <li>
            <i class="log"></i>log
          </li>
          <li>
            <i class="rock"></i>rock
          </li>
          <li>
            <i class="stump"></i>stump
          </li>
          <li>
            <i class="pickable"></i>pickable
          </li>
          <li>
            <i class="drop"></i>drop
          </li>
          <li>
            <i class="home"></i>home
          </li>
          <li>
            <i class="left"></i>what the bot had to leave behind
          </li>
          <li>
            <i class="stuck"></i>stuck recovery
          </li>
          <li>
            <i class="path"></i>planned path
          </li>
        </ul>
      </details>
    </section>
  );
}
export function Supplies() {
  return (
    <section class="card supplies" id="supplies">
      <div class="section-heading">
        <div>
          <span class="eyebrow">PREPARE &amp; DIRECT</span>
          <h2>Kit &amp; objectives</h2>
        </div>
      </div>{" "}
      <details id="goals" open>
        <summary>
          Goals <span id="goal-count"></span>
        </summary>
        <p>Keep an objective through failed attempts and bounded recovery.</p>
        <button id="goal-combat" type="button" disabled>
          Test a Greyling encounter and return home
        </button>
        <p id="goal-result" role="status"></p>
        <div id="goal-list"></div>
      </details>
      <details id="inventory">
        <summary>
          Inventory <span id="inv-summary">not yet seen</span>
        </summary>
        <div
          id="inv-grid"
          role="grid"
          aria-label="The bot's inventory, as laid out in the game"
        ></div>
      </details>
      <p id="storage-needed" role="status" hidden></p>
    </section>
  );
}

export function Places() {
  return (
    <section class="card" id="navigation">
      <div class="section-heading">
        <div>
          <span class="eyebrow">REMEMBER &amp; RETURN</span>
          <h2>Places &amp; storage</h2>
        </div>
      </div>{" "}
      <details id="places">
        <summary>
          Places &amp; zones <span id="place-count"></span>
        </summary>
        <div class="place-toolbar">
          <select id="place-list" aria-label="Saved places">
            <option value="">Choose a place</option>
          </select>
          <button id="place-new" type="button">
            Add place
          </button>
        </div>
        <form id="place-form">
          <div class="place-fields">
            <label>
              Name{" "}
              <input
                id="place-name"
                maxlength="64"
                required
                autocomplete="off"
              />
            </label>
            <label>
              Purpose{" "}
              <select id="place-kind">
                <option value="place">General</option>
                <option value="storage">Storage</option>
                <option value="cooking">Cooking</option>
                <option value="wood">Wood</option>
                <option value="stone">Stone</option>
                <option value="flint">Flint</option>
                <option value="berries">Berries</option>
                <option value="ore">Ore</option>
              </select>
            </label>
            <label>
              X <input id="place-x" type="number" step="0.1" required />
            </label>
            <label>
              Height (Y){" "}
              <input id="place-y" type="number" step="0.1" required />
            </label>
            <label>
              Z <input id="place-z" type="number" step="0.1" required />
            </label>
            <label>
              Zone radius (m){" "}
              <input
                id="place-radius"
                type="number"
                min="0"
                max="100"
                step="0.5"
                value="0"
                required
              />
            </label>
          </div>
          <div class="place-toolbar">
            <button id="place-bot" type="button">
              Use Tas’s position
            </button>
            <button id="place-pick" type="button" aria-pressed="false">
              Place on map
            </button>
            <label>
              <input id="place-ground" type="checkbox" /> Use terrain height
            </label>
          </div>
          <p class="place-help">
            Radius 0 marks a point. A larger radius marks a circular zone. Keep
            the height of raised floors when moving places inside a building.
          </p>
          <p id="place-source"></p>
          <div class="place-toolbar">
            <button type="submit">Save</button>
            <button id="place-go" type="button">
              Go to saved place
            </button>
            <button id="place-delete" type="button">
              Delete
            </button>
            <button id="place-near" type="button">
              What’s nearby?
            </button>
          </div>
        </form>
        <p id="place-result" role="status"></p>
        <div id="place-nearby"></div>
        <p class="place-help">
          Tas marks standard locations from things he observes. Your changes
          take precedence; deleted automatic places stay deleted. Nearby
          contents are last known.
        </p>
      </details>
    </section>
  );
}
export function Preferences() {
  return (
    <section class="card preferences" id="preferences">
      <div class="section-heading">
        <div>
          <span class="eyebrow">CONFIGURE</span>
          <h2>People &amp; preferences</h2>
        </div>
      </div>{" "}
      <details id="presence-controls">
        <summary>
          Players &amp; presence <span id="presence-summary"></span>
        </summary>
        <div class="presence-options">
          <label>
            <input id="share-position" type="checkbox" /> Share Tas’s map marker
            with the server
          </label>
          <span id="marker-state" role="status"></span>
          <label>
            <input id="marco-polo" type="checkbox" /> Marco/Polo greetings and
            location replies
          </label>
        </div>
        <p id="profile-owners"></p>
        <table id="profile-table" aria-label="Player control profiles"></table>
        <form id="profile-form" class="place-toolbar">
          <label>
            Player{" "}
            <input
              id="profile-name"
              maxlength="64"
              required
              autocomplete="off"
            />
          </label>
          <label>
            Profile{" "}
            <select id="profile-role">
              <option value="companion">Companion</option>
              <option value="guest">Guest</option>
              <option value="operator">Operator</option>
            </select>
          </label>
          <button type="submit">Assign profile</button>
        </form>
        <p class="place-help">
          Operator: commands, locating and join greeting. Companion: locating
          and greeting. Guest: locating only. Owners retain full authority.
        </p>
        <p class="place-help">
          Shout “tas Marco” for a Polo reply describing where you are relative
          to Tas. Reply “Polo” to his join greeting for directions to him.
          Distances are straight-line estimates.
        </p>
        <p id="presence-result" role="status"></p>
      </details>
      <details id="voice">
        <summary>
          Voice <span id="voice-summary"></span>
        </summary>
        <div class="voice-row">
          <label>
            Model <select id="model-select"></select>
          </label>
          <button id="model-use">Use</button>
          <button
            id="model-refresh"
            title="Ask the backend which models it offers"
          >
            Refresh
          </button>
          <span id="model-result" role="status"></span>
        </div>
        <p id="model-note"></p>
      </details>
      <details id="settings">
        <summary>
          Settings <span id="settings-summary"></span>
        </summary>
        <table id="settings-table"></table>
        <p id="settings-result" role="status"></p>
      </details>
    </section>
  );
}
export function CommandBar() {
  return (
    <form
      id="say-form"
      class="say-row command-bar"
      autocomplete="off"
      aria-label="Speak to Tas"
    >
      <label>
        Say as <select id="say-as"></select>
      </label>
      <input
        id="say-text"
        type="text"
        placeholder="a command, such as: follow me, go home, status"
        aria-label="What to say to the bot"
      />
      <button>Say</button>
      <span id="say-result" role="status"></span>
    </form>
  );
}
export function Timeline() {
  return (
    <section class="card timeline" id="feed-pane">
      <div class="section-heading">
        <div>
          <span class="eyebrow">UNDERSTAND</span>
          <h2>Activity</h2>
        </div>
        <span class="muted">Tasks &amp; chat · details on demand</span>
      </div>{" "}
      <details class="timeline-options">
        <summary>Filters &amp; history</summary>
        <div class="filters">
          <label>
            <input type="checkbox" data-cat="chat" checked /> Chat
          </label>
          <label>
            <input type="checkbox" data-cat="task" checked /> Tasks
          </label>
          <label>
            <input type="checkbox" data-cat="trace" /> Plugin decisions
          </label>
          <label>
            <input type="checkbox" data-cat="diagnostic" /> Diagnostics
          </label>
          <label>
            <input type="checkbox" data-cat="decision" /> Controller
          </label>
          <label>
            <input type="checkbox" data-cat="other" /> Other
          </label>
          <label>
            <input type="checkbox" data-cat="state" /> State
          </label>
          <label class="push">
            <input type="checkbox" id="pause" /> Pause
          </label>
          <select id="day" title="Watch live, or read a past day's audit log">
            <option value="">Live</option>
          </select>
        </div>
      </details>{" "}
      <ol id="feed" aria-label="Activity timeline"></ol>
      <ol id="history" class="feed" hidden></ol>
      <dialog id="task-view">
        <header>
          <h2 id="task-title"></h2>
          <button id="task-close" title="Close">
            Close
          </button>
        </header>
        <ol id="task-lines" class="feed"></ol>
      </dialog>
    </section>
  );
}
