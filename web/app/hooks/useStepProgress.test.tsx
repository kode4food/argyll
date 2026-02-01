import { renderHook } from "@testing-library/react";
import { useStepProgress } from "./useStepProgress";
import { ExecutionResult } from "../api";

const mockUseExecutions = jest.fn();

jest.mock("../store/flowStore", () => ({
  useExecutions: () => mockUseExecutions(),
}));

describe("useStepProgress", () => {
  beforeEach(() => {
    mockUseExecutions.mockReturnValue([]);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("returns pending status when no flowId provided", () => {
    const { result } = renderHook(() =>
      useStepProgress("step-1", undefined, undefined)
    );

    expect(result.current.status).toBe("pending");
    expect(result.current.flowId).toBeUndefined();
  });

  test("returns status from execution prop when provided", () => {
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "flow-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      outputs: { result: "value" },
    };

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", execution)
    );

    expect(result.current.status).toBe("completed");
    expect(result.current.flowId).toBe("flow-1");
  });

  test("uses store executions when execution prop is missing", () => {
    mockUseExecutions.mockReturnValue([
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        inputs: {},
        started_at: "2024-01-01T00:00:00Z",
      },
    ]);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("active");
  });

  test("ignores execution prop when flowId does not match", () => {
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "flow-different",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", execution)
    );

    expect(result.current.status).toBe("pending");
    expect(result.current.flowId).toBe("flow-1");
  });

  test("computes work item progress", () => {
    mockUseExecutions.mockReturnValue([
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        inputs: {},
        started_at: "2024-01-01T00:00:00Z",
        work_items: {
          a: { status: "succeeded" },
          b: { status: "failed" },
          c: { status: "active" },
        },
      },
    ]);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toEqual({
      total: 3,
      completed: 1,
      failed: 1,
      active: 1,
    });
  });
});
