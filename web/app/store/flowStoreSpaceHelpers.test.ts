import { assignStepSpaces } from "./flowStoreSpaceHelpers";

describe("assignStepSpaces", () => {
  test("adds, removes, and creates Space selections", () => {
    const kept = new Set(["other"]);
    const result = assignStepSpaces(
      { a: new Set(["s1"]), b: new Set(), c: kept },
      "s1",
      ["b", "d"]
    );
    expect(result.a).toEqual(new Set());
    expect(result.b).toEqual(new Set(["s1"]));
    expect(result.c).toBe(kept);
    expect(result.d).toEqual(new Set(["s1"]));
  });
});
