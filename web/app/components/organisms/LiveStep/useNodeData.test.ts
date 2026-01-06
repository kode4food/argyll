import { renderHook } from "@testing-library/react";
import { useNodeData } from "./useNodeData";
import { AttributeRole, AttributeType, ExecutionResult } from "@/app/api";

describe("useNodeData", () => {
  const mockStep = {
    id: "step-1",
    name: "Test Step",
    type: "sync" as const,
    attributes: {
      input1: {
        role: AttributeRole.Required,
        type: AttributeType.String,
        description: "",
      },
      output1: {
        role: AttributeRole.Output,
        type: AttributeType.String,
        description: "",
      },
    },
  };

  const mockExecution: ExecutionResult = {
    step_id: "step-1",
    flow_id: "flow-1",
    status: "completed",
    inputs: {},
    started_at: "2024-01-01T00:00:00Z",
    duration_ms: 100,
  };

  it("finds execution for the step", () => {
    const executions = [
      { ...mockExecution, id: "exec-1", step_id: "step-1" },
      { ...mockExecution, id: "exec-2", step_id: "step-2" },
    ];

    const { result } = renderHook(() =>
      useNodeData(mockStep as any, null, executions as any, [])
    );

    expect(result.current.execution?.id).toBe("exec-1");
  });

  it("returns undefined execution when not found", () => {
    const executions = [{ ...mockExecution, id: "exec-1", step_id: "step-2" }];

    const { result } = renderHook(() =>
      useNodeData(mockStep as any, null, executions as any, [])
    );

    expect(result.current.execution).toBeUndefined();
  });

  it("creates a Set from resolved attributes", () => {
    const resolved = ["input1", "input2"];

    const { result } = renderHook(() =>
      useNodeData(mockStep as any, null, [], resolved)
    );

    expect(result.current.resolved).toBeInstanceOf(Set);
    expect(result.current.resolved.has("input1")).toBe(true);
    expect(result.current.resolved.has("input2")).toBe(true);
  });

  it("builds provenance map from flow state", () => {
    const flowData = {
      id: "flow-1",
      state: {
        input1: { step: "step-1" },
        input2: { step: "step-2" },
      },
    };

    const { result } = renderHook(() =>
      useNodeData(mockStep as any, flowData as any, [], [])
    );

    expect(result.current.provenance).toBeInstanceOf(Map);
    expect(result.current.provenance.get("input1")).toBe("step-1");
    expect(result.current.provenance.get("input2")).toBe("step-2");
  });

  it("handles undefined flow data", () => {
    const { result } = renderHook(() =>
      useNodeData(mockStep as any, undefined, [], [])
    );

    expect(result.current.provenance).toBeInstanceOf(Map);
    expect(result.current.provenance.size).toBe(0);
  });

  it("calculates satisfied arguments", () => {
    const { result } = renderHook(() =>
      useNodeData(mockStep as any, null, [], ["input1"])
    );

    expect(result.current.satisfied).toBeInstanceOf(Set);
    expect(result.current.satisfied.has("input1")).toBe(true);
    expect(result.current.satisfied.has("output1")).toBe(false);
  });

  it("returns all required data structures", () => {
    const { result } = renderHook(() =>
      useNodeData(mockStep as any, null, [], [])
    );

    expect(result.current).toHaveProperty("execution");
    expect(result.current).toHaveProperty("resolved");
    expect(result.current).toHaveProperty("provenance");
    expect(result.current).toHaveProperty("satisfied");
  });

  it("memoizes resolved set", () => {
    const resolved = ["input1"];
    const { result, rerender } = renderHook(
      ({ resolved: r }) => useNodeData(mockStep as any, null, [], r),
      { initialProps: { resolved } }
    );

    const firstResolved = result.current.resolved;

    rerender({ resolved });

    // Should be the same object due to memoization
    expect(result.current.resolved).toBe(firstResolved);
  });

  it("updates resolved set when it changes", () => {
    const { result, rerender } = renderHook(
      ({ resolved }) => useNodeData(mockStep as any, null, [], resolved),
      { initialProps: { resolved: ["input1"] } }
    );

    expect(result.current.resolved.has("input1")).toBe(true);

    rerender({ resolved: ["input2"] });

    expect(result.current.resolved.has("input1")).toBe(false);
    expect(result.current.resolved.has("input2")).toBe(true);
  });

  it("memoizes provenance map", () => {
    const flowData = {
      id: "flow-1",
      state: { input1: { step: "step-1" } },
    };

    const { result, rerender } = renderHook(() =>
      useNodeData(mockStep as any, flowData as any, [], [])
    );

    const firstProvenance = result.current.provenance;

    rerender();

    expect(result.current.provenance).toBe(firstProvenance);
  });

  it("updates provenance when flow state changes", () => {
    const { result, rerender } = renderHook(
      ({ flowData }) => useNodeData(mockStep as any, flowData as any, [], []),
      {
        initialProps: {
          flowData: {
            id: "flow-1",
            state: { input1: { step: "step-1" } },
          },
        },
      }
    );

    expect(result.current.provenance.get("input1")).toBe("step-1");

    rerender({
      flowData: {
        id: "flow-1",
        state: { input1: { step: "step-2" } },
      },
    });

    expect(result.current.provenance.get("input1")).toBe("step-2");
  });

  it("handles default parameters for executions and resolvedAttributes", () => {
    const { result } = renderHook(() => useNodeData(mockStep as any, null));

    expect(result.current.execution).toBeUndefined();
    expect(result.current.resolved).toBeInstanceOf(Set);
    expect(result.current.resolved.size).toBe(0);
    expect(result.current.satisfied).toBeInstanceOf(Set);
  });

  it("handles null flow data with attributes", () => {
    const { result } = renderHook(() => useNodeData(mockStep as any, null));

    expect(result.current.provenance).toBeInstanceOf(Map);
    expect(result.current.satisfied.size).toBeGreaterThanOrEqual(0);
  });

  it("handles satisfied args with multiple attributes", () => {
    const stepWithMultipleAttrs = {
      ...mockStep,
      attributes: {
        input1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        input2: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        optional1: {
          role: AttributeRole.Optional,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const { result } = renderHook(() =>
      useNodeData(
        stepWithMultipleAttrs as any,
        null,
        [],
        ["input1", "optional1"]
      )
    );

    expect(result.current.satisfied.has("input1")).toBe(true);
    expect(result.current.satisfied.has("input2")).toBe(false);
    expect(result.current.satisfied.has("optional1")).toBe(true);
  });
});
