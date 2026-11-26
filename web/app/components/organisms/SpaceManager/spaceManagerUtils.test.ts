import { Space } from "@/app/api";
import { emptyDraft, seedDraft, toQBE } from "./spaceManagerUtils";

describe("spaceManagerUtils", () => {
  test("seeds a new Space from what a chat draft gives", () => {
    expect(seedDraft({})).toEqual(emptyDraft());
    expect(seedDraft({ name: "Risk", qbe: [["domain:risk"]] })).toMatchObject({
      name: "Risk",
      qbe: [["domain:risk"]],
      scriptMode: false,
    });
    expect(
      seedDraft({ selector: { language: "lua", source: "return true" } })
    ).toMatchObject({ qbe: [[]], scriptMode: true });
  });

  test("seeds defaults for fields a chat draft sends as null", () => {
    const seed = { id: null, qbe: null } as unknown as Partial<Space>;
    expect(seedDraft(seed)).toEqual(emptyDraft());
  });

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
