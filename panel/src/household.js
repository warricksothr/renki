export const householdFoods = [
  ["Mushroom", "Mushrooms"],
  ["MushroomYellow", "Yellow mushrooms"],
  ["Raspberry", "Raspberries"],
  ["Blueberries", "Blueberries"],
  ["Honey", "Honey"],
  ["CookedMeat", "Cooked boar meat"],
  ["CookedDeerMeat", "Cooked deer meat"],
  ["NeckTailGrilled", "Grilled neck tails"],
];

export function householdTargets(rows) {
  if (!rows.length || rows.length > 8)
    throw Error("Choose one to eight foods for this household plan.");
  const targets = {},
    minimum = {};
  for (const row of rows) {
    const count = Number(row.count),
      below = Number(row.minimum);
    if (!householdFoods.some(([id]) => id === row.food))
      throw Error("Choose a supported food for each row.");
    if (Object.hasOwn(targets, row.food))
      throw Error("Each food can appear only once in a household plan.");
    if (!Number.isInteger(count) || count < 1 || count > 20)
      throw Error("Each food needs a refill target of 1–20 meals.");
    if (!Number.isInteger(below) || below < 1 || below > count)
      throw Error("Each refill minimum must be between one and its target.");
    targets[row.food] = count;
    minimum[row.food] = below;
  }
  return { targets, minimum };
}
