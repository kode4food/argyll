import { renderHook, waitFor } from "@testing-library/react";
import { useLayoutPlan } from "./useLayoutPlan";
import { generateOverviewPlan, shouldApplyAutoLayout } from "./layoutUtils";
import { saveNodePositions } from "@/utils/nodePositioning";
import { Step } from "@/app/api";

jest.mock("./layoutUtils", () => ({
  generateOverviewPlan: jest.fn(),
  shouldApplyAutoLayout: jest.fn(),
}));

jest.mock("@/utils/nodePositioning", () => ({
  saveNodePositions: jest.fn(),
}));

describe("useLayoutPlan", () => {
  const shouldApplyAutoLayoutMock = shouldApplyAutoLayout as jest.Mock;
  const generateOverviewPlanMock = generateOverviewPlan as jest.Mock;
  const saveNodePositionsMock = saveNodePositions as jest.Mock;
  const steps: Step[] = [
    { id: "step-1", name: "Step 1", type: "sync", attributes: {} },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("returns null plan when auto layout is disabled", () => {
    shouldApplyAutoLayoutMock.mockReturnValue(false);

    const { result } = renderHook(() => useLayoutPlan(steps, []));

    expect(result.current.plan).toBeNull();
    expect(generateOverviewPlanMock).not.toHaveBeenCalled();
  });

  test("saves node positions when plan exists and nodes are arranged", async () => {
    const plan = { goals: [], required: [], steps: {}, attributes: {} };
    shouldApplyAutoLayoutMock.mockReturnValue(true);
    generateOverviewPlanMock.mockReturnValue(plan);
    const arrangedNodes = [
      { id: "step-1", position: { x: 0, y: 0 }, data: {}, type: "stepNode" },
    ];

    const { result } = renderHook(() => useLayoutPlan(steps, arrangedNodes));

    expect(result.current.plan).toBe(plan);
    await waitFor(() => {
      expect(saveNodePositionsMock).toHaveBeenCalledWith(arrangedNodes);
    });
  });
});
