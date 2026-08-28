import { renderHook } from "@testing-library/react";
import { useNodeCalculation } from "./useNodeCalculation";
import { AttributeRole, FlowContext, Step } from "@/app/api";
import { stepLayout } from "@/constants/layout";
import { loadNodePositions } from "@/utils/nodePositioning";

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(),
}));

describe("useNodeCalculation", () => {
  const loadNodePositionsMock = loadNodePositions as jest.Mock;
  const savedPositionX = 10;
  const savedPositionY = 20;
  const flowData: FlowContext = {
    id: "flow-1",
    status: "active",
    state: {},
    started_at: "2024-01-01T00:00:00Z",
    plan: {
      goals: ["step-2"],
      required: [],
      steps: {},
      attributes: {},
    },
  };
  const executions = [{ step_id: "step-1" }] as any[];
  const resolvedAttributes = ["data"];
  const diagramContainerRef = { current: null };

  const sectionHeightFor = (count: number) => {
    if (count === 0) return 0;
    return stepLayout.SECTION_HEIGHT + count * stepLayout.ARG_LINE_HEIGHT;
  };

  beforeEach(() => {
    jest.clearAllMocks();
    loadNodePositionsMock.mockReturnValue({});
  });

  test("uses saved positions when available", () => {
    const savedPosition = { x: savedPositionX, y: savedPositionY };
    const steps: Step[] = [
      { id: "step-1", name: "Step 1", type: "service", attributes: {} },
    ];

    loadNodePositionsMock.mockReturnValue({ "step-1": savedPosition });

    const { result } = renderHook(() => useNodeCalculation(steps, flowData));

    expect(loadNodePositionsMock).toHaveBeenCalledWith({
      type: "flow",
      flowId: "flow-1",
    });
    expect(result.current[0].position).toEqual(savedPosition);
  });

  test("sets dependency metadata and levels", () => {
    const steps: Step[] = [
      {
        id: "step-1",
        name: "Step 1",
        type: "service",
        attributes: { data: { role: AttributeRole.Output } },
      },
      {
        id: "step-2",
        name: "Step 2",
        type: "service",
        attributes: { data: { role: AttributeRole.Required } },
      },
    ];

    const { result } = renderHook(() =>
      useNodeCalculation(
        steps,
        flowData,
        executions,
        resolvedAttributes,
        diagramContainerRef
      )
    );

    expect(loadNodePositionsMock).toHaveBeenCalledWith({
      type: "flow",
      flowId: "flow-1",
    });
    const [first, second] = result.current;

    expect(first.data.isStartingPoint).toBe(true);
    expect(second.data.isStartingPoint).toBe(false);
    expect(first.data.isGoalStep).toBe(false);
    expect(second.data.isGoalStep).toBe(true);
    expect(second.data.executions).toBe(executions);
    expect(second.data.resolvedAttributes).toBe(resolvedAttributes);
    expect(second.data.diagramContainerRef).toBe(diagramContainerRef);

    expect(first.position).toEqual({
      x: 0,
      y: stepLayout.VERTICAL_OFFSET,
    });
    expect(second.position).toEqual({
      x: stepLayout.HORIZONTAL_SPACING,
      y: stepLayout.VERTICAL_OFFSET,
    });
  });

  test("stacks nodes within the same level", () => {
    const requiredCount = 1;
    const outputCount = 2;
    const half = 0.5;
    const steps: Step[] = [
      {
        id: "step-1",
        name: "Step 1",
        type: "service",
        attributes: { input: { role: AttributeRole.Required } },
      },
      {
        id: "step-2",
        name: "Step 2",
        type: "service",
        attributes: {
          outputA: { role: AttributeRole.Output },
          outputB: { role: AttributeRole.Output },
        },
      },
    ];

    const { result } = renderHook(() => useNodeCalculation(steps));

    const firstHeight =
      stepLayout.WIDGET_BASE_HEIGHT + sectionHeightFor(requiredCount);
    const secondHeight =
      stepLayout.WIDGET_BASE_HEIGHT + sectionHeightFor(outputCount);
    const rowOffset = (stepLayout.VERTICAL_SPACING + firstHeight) * half;
    const rowOffsetSecond = (stepLayout.VERTICAL_SPACING + secondHeight) * half;

    expect(result.current[0].position).toEqual({
      x: 0,
      y: stepLayout.VERTICAL_OFFSET - rowOffset,
    });
    expect(result.current[1].position).toEqual({
      x: 0,
      y: stepLayout.VERTICAL_OFFSET + rowOffsetSecond,
    });
  });
});
