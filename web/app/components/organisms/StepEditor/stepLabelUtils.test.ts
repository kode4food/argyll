import { buildLabelsFromStep, createStepLabels } from "./stepLabelUtils";
import { Label } from "./stepEditorTypes";
import { Step } from "@/app/api";

const label = (overrides: Partial<Label> = {}): Label => ({
  id: "label-1",
  key: "team",
  value: "risk",
  ...overrides,
});

const step = (labels?: Record<string, string>): Step => ({
  id: "test-step",
  name: "Test",
  type: "sync",
  attributes: {},
  labels,
});

describe("stepLabelUtils", () => {
  test("builds label rows sorted by key", () => {
    expect(
      buildLabelsFromStep(step({ team: "risk", domain: "trading" }))
    ).toEqual([
      expect.objectContaining({ key: "domain", value: "trading" }),
      expect.objectContaining({ key: "team", value: "risk" }),
    ]);
  });

  test("builds no rows without labels", () => {
    expect(buildLabelsFromStep(step())).toEqual([]);
    expect(buildLabelsFromStep(null)).toEqual([]);
  });

  test("creates trimmed step labels", () => {
    const result = createStepLabels([
      label({ key: " team ", value: " risk " }),
      label({ id: "label-2", key: "empty", value: "" }),
    ]);

    expect(result).toEqual({ team: "risk", empty: "" });
  });

  test("skips rows with a blank key", () => {
    expect(createStepLabels([label({ key: "   " })])).toBeUndefined();
  });

  test("creates no labels when none are present", () => {
    expect(createStepLabels([])).toBeUndefined();
  });

  test("keeps the last value for a duplicate key", () => {
    const result = createStepLabels([
      label(),
      label({ id: "label-2", value: "trading" }),
    ]);

    expect(result).toEqual({ team: "trading" });
  });
});
