import { AttributeRole, Step } from "@/app/api";
import {
  generateOverviewPlan,
  hasSavedPositions,
  shouldApplyAutoLayout,
} from "./layoutUtils";
import { loadNodePositions } from "@/utils/nodePositioning";

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(),
}));

describe("layoutUtils", () => {
  const loadNodePositionsMock = loadNodePositions as jest.Mock;
  const steps: Step[] = [
    { id: "s1", name: "Step 1", type: "sync", attributes: {} },
    { id: "s2", name: "Step 2", type: "sync", attributes: {} },
  ];
  const position = { x: 10, y: 20 };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("generateOverviewPlan returns null with no steps", () => {
    expect(generateOverviewPlan([])).toBeNull();
  });

  test("generateOverviewPlan builds providers and consumers", () => {
    const planSteps: Step[] = [
      {
        id: "s1",
        name: "Step 1",
        type: "sync",
        attributes: { data: { role: AttributeRole.Output } },
      },
      {
        id: "s2",
        name: "Step 2",
        type: "sync",
        attributes: {
          data: { role: AttributeRole.Required },
          extra: { role: AttributeRole.Optional },
        },
      },
    ];

    const plan = generateOverviewPlan(planSteps);

    expect(plan).not.toBeNull();
    expect(plan?.attributes.data).toEqual({
      providers: ["s1"],
      consumers: ["s2"],
    });
    expect(plan?.attributes.extra).toEqual({
      providers: [],
      consumers: ["s2"],
    });
    expect(plan?.steps.s1).toBe(planSteps[0]);
  });

  test("hasSavedPositions returns true when any step has saved position", () => {
    loadNodePositionsMock.mockReturnValue({ s2: position });
    expect(hasSavedPositions(steps)).toBe(true);
  });

  test("shouldApplyAutoLayout returns false with no steps", () => {
    expect(shouldApplyAutoLayout([])).toBe(false);
  });

  test("shouldApplyAutoLayout returns false when saved positions exist", () => {
    loadNodePositionsMock.mockReturnValue({ s1: position });
    expect(shouldApplyAutoLayout(steps)).toBe(false);
  });

  test("shouldApplyAutoLayout returns true when no saved positions exist", () => {
    loadNodePositionsMock.mockReturnValue({});
    expect(shouldApplyAutoLayout(steps)).toBe(true);
  });
});
