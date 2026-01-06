import { renderHook } from "@testing-library/react";
import { useNodeCalculation } from "./useNodeCalculation";
import type { Step, FlowContext, ExecutionResult } from "@/app/api";
import { AttributeRole, AttributeType } from "@/app/api";

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(() => ({})),
}));

describe("useNodeCalculation", () => {
  const createStep = (id: string): Step => ({
    id,
    name: `Step ${id}`,
    type: "sync",
    attributes: {
      out: { role: AttributeRole.Output, type: AttributeType.String },
    },
    http: {
      endpoint: "http://test",
      timeout: 5000,
    },
  });

  test("includes flow data and execution details", () => {
    const step = createStep("step1");
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
      plan: {
        goals: ["step1"],
        required: [],
        steps: { step1: step },
        attributes: {},
      },
    };
    const executions: ExecutionResult[] = [
      {
        step_id: "step1",
        flow_id: "wf-1",
        status: "completed",
        inputs: {},
        started_at: "2024-01-01T00:00:00Z",
      },
    ];

    const { result } = renderHook(() =>
      useNodeCalculation([step], flowData, executions)
    );

    expect(result.current[0].data.flowData).toEqual(flowData);
    expect(result.current[0].data.executions).toEqual(executions);
    expect(result.current[0].data.isGoalStep).toBe(true);
  });
});
