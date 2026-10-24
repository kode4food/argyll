import {
  spaceSelectorCompletions,
  stepAttributeCompletions,
} from "./scriptCompletions";

test("builds completions from Step attributes and selector metadata", () => {
  const attributes = [
    { name: "amount", role: "required", mappingName: "value" },
    { name: "result", role: "output" },
  ];
  expect(stepAttributeCompletions(attributes, false)).toEqual(["value"]);
  expect(stepAttributeCompletions(attributes, true)).toEqual([
    "value",
    "result",
  ]);

  const completions = spaceSelectorCompletions(
    [{ attributes: { score: {} }, tags: ["domain:risk"] }],
    "lua"
  );
  expect(completions).toContain('value["tags"]');
  expect(completions).toContain('value["attributes"]["score"]["compensated"]');
  expect(completions).toContain('has(value["tags"], "domain:risk")');

  const argExCompletions = spaceSelectorCompletions(
    [{ attributes: { score: {} }, tags: ["domain:risk"] }],
    "argyll"
  );
  expect(argExCompletions).toContain('"domain:risk" in value.tags');
  expect(argExCompletions).toContain("value.attributes.score.compensated");
});
