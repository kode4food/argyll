import { renderHook } from "@testing-library/react";
import { useLayoutPlan } from "./useLayoutPlan";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./layoutUtils";
import { Step } from "@/app/api";

jest.mock("./layoutUtils", () => ({
  generateOverviewPlan: jest.fn(),
  shouldApplyAutoLayout: jest.fn(),
}));

describe("useLayoutPlan", () => {
  const shouldApplyAutoLayoutMock = shouldApplyAutoLayout as jest.Mock;
  const generateOverviewPlanMock = generateOverviewPlan as jest.Mock;
  const steps: Step[] = [
    { id: "step-1", name: "Step 1", type: "service", attributes: {} },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("returns null plan when auto layout is disabled", () => {
    shouldApplyAutoLayoutMock.mockReturnValue(false);

    const { result } = renderHook(() => useLayoutPlan(steps));

    expect(result.current.plan).toBeNull();
    expect(generateOverviewPlanMock).not.toHaveBeenCalled();
  });

  test("returns the overview plan when auto layout applies", () => {
    const plan = {
      goals: { steps: [] },
      required: [],
      steps: {},
      attributes: {},
    };
    shouldApplyAutoLayoutMock.mockReturnValue(true);
    generateOverviewPlanMock.mockReturnValue(plan);

    const { result } = renderHook(() => useLayoutPlan(steps));

    expect(result.current.plan).toBe(plan);
  });
});
