import { renderHook } from "@testing-library/react";
import { useStepProgress } from "./useStepProgress";
import { ExecutionResult } from "../api";

const mockUseWebSocketContext = jest.fn();
const mockUseExecutions = jest.fn();

jest.mock("./useWebSocketContext", () => ({
  useWebSocketContext: () => mockUseWebSocketContext(),
}));

jest.mock("../store/flowStore", () => ({
  useExecutions: () => mockUseExecutions(),
}));

describe("useStepProgress", () => {
  beforeEach(() => {
    mockUseWebSocketContext.mockReturnValue({ events: [] });
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

  test("ignores execution prop when stepId does not match", () => {
    const execution: ExecutionResult = {
      step_id: "step-different",
      flow_id: "flow-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", execution)
    );

    expect(result.current.status).toBe("pending");
  });

  test("updates status from step_started event", () => {
    const events = [
      {
        type: "step_started",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
          start_time: 1234567890,
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("active");
    expect(result.current.startTime).toBe(1234567890);
    expect(result.current.flowId).toBe("flow-1");
  });

  test("updates status from step_completed event", () => {
    const events = [
      {
        type: "step_completed",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
          start_time: 1234567890,
          end_time: 1234567900,
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("completed");
    expect(result.current.startTime).toBe(1234567890);
    expect(result.current.endTime).toBe(1234567900);
    expect(result.current.flowId).toBe("flow-1");
  });

  test("updates status from step_failed event", () => {
    const events = [
      {
        type: "step_failed",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
          start_time: 1234567890,
          end_time: 1234567900,
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("failed");
    expect(result.current.startTime).toBe(1234567890);
    expect(result.current.endTime).toBe(1234567900);
  });

  test("uses latest event when multiple events exist", () => {
    const events = [
      {
        type: "step_started",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
          start_time: 1234567890,
        },
      },
      {
        type: "step_completed",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
          start_time: 1234567890,
          end_time: 1234567900,
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("completed");
  });

  test("ignores events for different step", () => {
    const events = [
      {
        type: "step_completed",
        data: {
          step_id: "step-different",
          flow_id: "flow-1",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("pending");
  });

  test("ignores events for different flow", () => {
    const events = [
      {
        type: "step_completed",
        data: {
          step_id: "step-1",
          flow_id: "flow-different",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("pending");
  });

  test("handles unknown event type as pending", () => {
    const events = [
      {
        type: "unknown_event_type",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("pending");
  });

  test("calculates work item progress when executions available", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        work_items: {
          "item-1": { id: "item-1", status: "completed" },
          "item-2": { id: "item-2", status: "active" },
          "item-3": { id: "item-3", status: "failed" },
          "item-4": { id: "item-4", status: "completed" },
        },
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toEqual({
      total: 4,
      completed: 2,
      failed: 1,
      active: 1,
    });
  });

  test("returns undefined work items when no flowId", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        work_items: {
          "item-1": { id: "item-1", status: "completed" },
        },
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const { result } = renderHook(() =>
      useStepProgress("step-1", undefined, undefined)
    );

    expect(result.current.workItems).toBeUndefined();
  });

  test("returns undefined work items when no executions", () => {
    mockUseExecutions.mockReturnValue(null);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toBeUndefined();
  });

  test("returns undefined work items when execution not found", () => {
    const executions = [
      {
        step_id: "step-different",
        flow_id: "flow-1",
        status: "active",
        work_items: {
          "item-1": { id: "item-1", status: "completed" },
        },
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toBeUndefined();
  });

  test("returns undefined work items when execution has no work items", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toBeUndefined();
  });

  test("returns undefined work items when work items is empty object", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        work_items: {},
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toBeUndefined();
  });

  test("includes work items in progress state from execution", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        work_items: {
          "item-1": { id: "item-1", status: "completed" },
        },
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "flow-1",
      status: "active",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", execution)
    );

    expect(result.current.workItems).toEqual({
      total: 1,
      completed: 1,
      failed: 0,
      active: 0,
    });
  });

  test("includes work items in progress state from events", () => {
    const executions = [
      {
        step_id: "step-1",
        flow_id: "flow-1",
        status: "active",
        work_items: {
          "item-1": { id: "item-1", status: "completed" },
          "item-2": { id: "item-2", status: "active" },
        },
      },
    ];
    mockUseExecutions.mockReturnValue(executions);

    const events = [
      {
        type: "step_started",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events });

    const { result } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.workItems).toEqual({
      total: 2,
      completed: 1,
      failed: 0,
      active: 1,
    });
  });

  test("updates when events change", () => {
    const events1 = [
      {
        type: "step_started",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events: events1 });

    const { result, rerender } = renderHook(() =>
      useStepProgress("step-1", "flow-1", undefined)
    );

    expect(result.current.status).toBe("active");

    const events2 = [
      ...events1,
      {
        type: "step_completed",
        data: {
          step_id: "step-1",
          flow_id: "flow-1",
        },
      },
    ];
    mockUseWebSocketContext.mockReturnValue({ events: events2 });

    rerender();

    expect(result.current.status).toBe("completed");
  });

  test("updates when execution changes", () => {
    const execution1: ExecutionResult = {
      step_id: "step-1",
      flow_id: "flow-1",
      status: "active",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result, rerender } = renderHook(
      ({ exec }) => useStepProgress("step-1", "flow-1", exec),
      { initialProps: { exec: execution1 } }
    );

    expect(result.current.status).toBe("active");

    const execution2: ExecutionResult = {
      step_id: "step-1",
      flow_id: "flow-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    rerender({ exec: execution2 });

    expect(result.current.status).toBe("completed");
  });
});
