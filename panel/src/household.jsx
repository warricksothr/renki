import { useState } from "preact/hooks";
import { householdFoods } from "./household.js";

export function HouseholdFoods() {
  const [rows, setRows] = useState([{ id: 0, food: "Mushroom" }]);
  const [serial, setSerial] = useState(1);
  return (
    <fieldset class="household-foods">
      <legend>Foods in this household plan</legend>
      <p>
        Each food has its own storage target. Tas works toward the whole mix.
      </p>
      <div id="household-food-rows">
        {rows.map((row, index) => (
          <div class="household-food-row" data-food-row key={row.id}>
            <label>
              Food to stock
              <select
                id={index === 0 ? "goal-stock-food" : undefined}
                data-food
                defaultValue={row.food}
              >
                {householdFoods.map(([id, name]) => (
                  <option value={id} key={id}>
                    {name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Refill to (meals in storage)
              <input
                id={index === 0 ? "goal-stock-count" : undefined}
                data-count
                type="number"
                min="1"
                max="20"
                defaultValue="4"
              />
            </label>
            <label>
              Refill below (meals)
              <input
                id={index === 0 ? "goal-stock-minimum" : undefined}
                data-minimum
                type="number"
                min="1"
                max="20"
                defaultValue="2"
              />
            </label>
            <button
              type="button"
              disabled={rows.length === 1}
              onClick={() => setRows(rows.filter((r) => r.id !== row.id))}
              aria-label={`Remove food row ${index + 1}`}
            >
              Remove
            </button>
          </div>
        ))}
      </div>
      <button
        id="household-add-food"
        type="button"
        disabled={rows.length >= 8}
        onClick={() => {
          const selected = new Set(
            [...document.querySelectorAll("[data-food-row] [data-food]")].map(
              (el) => el.value,
            ),
          );
          const food = householdFoods.find(([id]) => !selected.has(id))?.[0];
          if (food) {
            setRows([...rows, { id: serial, food }]);
            setSerial(serial + 1);
          }
        }}
      >
        Add food
      </button>
      <p class="place-help">
        Listed foods use the existing forage and basic cooking routines. Other
        recipes need their own supported preparation routine.
      </p>
    </fieldset>
  );
}
