import { toQBE } from "./spaceManagerUtils";

describe("spaceManagerUtils", () => {
  test("sorts and dedupes within a term", () => {
    expect(toQBE([["b", " a ", "b", ""]])).toEqual([["a", "b"]]);
  });

  test("drops terms left empty", () => {
    expect(toQBE([[], ["a"], ["  "]])).toEqual([["a"]]);
  });

  // The engine orders terms with slices.Compare, so a term that is a prefix
  // of another sorts first. A mismatch here silently breaks the preview and
  // dirty checks, which compare this ordering against the engine's
  test("orders terms element-wise, shorter prefix first", () => {
    expect(toQBE([["a", "b"], ["b"], ["a"]])).toEqual([
      ["a"],
      ["a", "b"],
      ["b"],
    ]);
  });

  test("dedupes equal terms", () => {
    expect(
      toQBE([
        ["b", "a"],
        ["a", "b"],
      ])
    ).toEqual([["a", "b"]]);
  });
});
