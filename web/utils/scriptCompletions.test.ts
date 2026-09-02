import {
  spaceSelectorCompletions,
  stepAttributeCompletions,
} from "./scriptCompletions";

test("builds completions from Step attributes and selector metadata", () => {
  const attributes = [
    { name: "amount", role: "required", mappingName: "value" },
    { name: "result", role: "output" },
  ];
  expect(stepAttributeCompletions(attributes, "lua", false)).toEqual(["value"]);
  expect(stepAttributeCompletions(attributes, "lua", true)).toEqual([
    "value",
    "result",
  ]);
  expect(stepAttributeCompletions(attributes, "jpath", false)).toEqual([
    '$["value"]',
  ]);

  const completions = spaceSelectorCompletions(
    [{ attributes: { score: {} }, tags: ["domain:risk"] }],
    "lua"
  );
  expect(completions).toContain('value["tags"]["domain:risk"]');
  expect(completions).toContain('value["attributes"]["score"]["compensated"]');
});
