import { fromGoalSets, toGoalSets } from "./goals";

describe("goals", () => {
  test("converts between a goal chain and its sets", () => {
    const chain = { steps: ["a", "b"], else: { steps: ["c"] } };

    expect(toGoalSets(chain)).toEqual([["a", "b"], ["c"]]);
    expect(fromGoalSets([["a", "b"], ["c"]])).toEqual(chain);
  });

  test("handles missing goals", () => {
    expect(toGoalSets(undefined)).toEqual([]);
    expect(fromGoalSets([])).toEqual({ steps: [] });
  });
});
