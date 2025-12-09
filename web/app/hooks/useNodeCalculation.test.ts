import { renderHook } from "@testing-library/react";
import { useNodeCalculation } from "./useNodeCalculation";
import type { Step } from "../api";
import { AttributeRole, AttributeType } from "../api";

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(() => ({})),
}));

describe("useNodeCalculation", () => {
  const createStep = (
    id: string,
    requiredArgs: number = 0,
    optionalArgs: number = 0,
    outputs: number = 0
  ): Step => {
    const attributes: Record<string, import("@/app/api").AttributeSpec> = {};
    for (let i = 0; i < requiredArgs; i++) {
      attributes[`req${i}`] = {
        role: AttributeRole.Required,
        type: AttributeType.String,
      };
    }
    for (let i = 0; i < optionalArgs; i++) {
      attributes[`opt${i}`] = {
        role: AttributeRole.Optional,
        type: AttributeType.String,
      };
    }
    for (let i = 0; i < outputs; i++) {
      attributes[`out${i}`] = {
        role: AttributeRole.Output,
        type: AttributeType.String,
      };
    }

    return {
      id,
      name: `Step ${id}`,
      type: "sync",
      attributes,
      version: "1.0.0",
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };
  };

  test("returns empty array for no steps", () => {
    const { result } = renderHook(() => useNodeCalculation([], []));

    expect(result.current).toEqual([]);
  });

  test("creates nodes for all visible steps", () => {
    const steps = [createStep("step1"), createStep("step2")];

    const { result } = renderHook(() => useNodeCalculation(steps, []));

    expect(result.current).toHaveLength(2);
    expect(result.current[0].id).toBe("step1");
    expect(result.current[1].id).toBe("step2");
  });

  test("assigns correct step data to nodes", () => {
    const step = createStep("step1");

    const { result } = renderHook(() => useNodeCalculation([step], []));

    expect(result.current[0].data.step).toEqual(step);
  });

  test("sets selected state when step matches selectedStep", () => {
    const step = createStep("step1");

    const { result } = renderHook(() => useNodeCalculation([step], ["step1"]));

    expect(result.current[0].data.selected).toBe(true);
  });

  test("does not set selected when step does not match", () => {
    const step = createStep("step1");

    const { result } = renderHook(() => useNodeCalculation([step], ["step2"]));

    expect(result.current[0].data.selected).toBe(false);
  });

  test("calculates positions based on dependency levels", () => {
    const step1 = createStep("step1", 0, 0, 1);
    const step2 = {
      ...createStep("step2", 1, 0, 0),
      attributes: {
        out0: { role: AttributeRole.Required, type: AttributeType.String },
      },
    };

    const { result } = renderHook(() => useNodeCalculation([step1, step2], []));

    const node1 = result.current.find((n) => n.id === "step1");
    const node2 = result.current.find((n) => n.id === "step2");

    expect(node1?.position.x).toBeLessThan(node2?.position.x || 0);
  });

  test("marks goal step in flow data", () => {
    const step = createStep("step1");
    const flowData: any = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "",
      plan: {
        goals: ["step1"],
        steps: {
          step1: { step: step },
        },
        required: [],
        attributes: {},
      },
    };

    const { result } = renderHook(() =>
      useNodeCalculation([step], [], flowData)
    );

    expect(result.current[0].data.isGoalStep).toBe(true);
  });

  test("marks goal step in preview plan", () => {
    const step = createStep("step1");
    const previewPlan: any = {
      goals: ["step1"],
      steps: {
        step1: { step: step },
      },
      required: [],
    };

    const { result } = renderHook(() =>
      useNodeCalculation([step], [], undefined, undefined, previewPlan)
    );

    expect(result.current[0].data.isGoalStep).toBe(true);
  });

  test("sets isInPreviewPlan correctly", () => {
    const step1 = createStep("step1");
    const step2 = createStep("step2");
    const previewStepIds = new Set(["step1"]);

    const { result } = renderHook(() =>
      useNodeCalculation(
        [step1, step2],
        [],
        undefined,
        undefined,
        undefined,
        previewStepIds
      )
    );

    const node1 = result.current.find((n) => n.id === "step1");
    const node2 = result.current.find((n) => n.id === "step2");

    expect(node1?.data.isInPreviewPlan).toBe(true);
    expect(node2?.data.isInPreviewPlan).toBe(false);
  });

  test("sets isPreviewMode when preview plan exists", () => {
    const step = createStep("step1");
    const previewPlan: any = {
      goals: [],
      steps: {},
      required: [],
    };

    const { result } = renderHook(() =>
      useNodeCalculation([step], [], undefined, undefined, previewPlan)
    );

    expect(result.current[0].data.isPreviewMode).toBe(true);
  });

  test("identifies starting points in execution plan", () => {
    const step1 = createStep("step1", 0, 0, 1);
    const step2 = {
      ...createStep("step2", 1, 0, 0),
      attributes: {
        out0: { role: AttributeRole.Required, type: AttributeType.String },
      },
    };

    const flowData: any = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "",
      plan: {
        goals: ["step2"],
        steps: {
          step1: { step: step1 },
          step2: { step: step2 },
        },
        required: [],
        attributes: {},
      },
    };

    const { result } = renderHook(() =>
      useNodeCalculation([step1, step2], [], flowData)
    );

    const node1 = result.current.find((n) => n.id === "step1");
    const node2 = result.current.find((n) => n.id === "step2");

    expect(node1?.data.isStartingPoint).toBe(true);
    expect(node2?.data.isStartingPoint).toBe(false);
  });

  test("calls onStepClick when node is clicked", () => {
    const step = createStep("step1");
    const onStepClick = jest.fn();

    const { result } = renderHook(() =>
      useNodeCalculation(
        [step],
        [],
        undefined,
        undefined,
        undefined,
        undefined,
        onStepClick
      )
    );

    const node = result.current[0] as any;
    node.data.onStepClick("step1");

    expect(onStepClick).toHaveBeenCalledWith("step1");
  });

  test("does not crash when onStepClick is not provided", () => {
    const step = createStep("step1");

    const { result } = renderHook(() => useNodeCalculation([step], []));

    const node = result.current[0] as any;
    expect(() => node.data.onStepClick?.("step1")).not.toThrow();
  });

  test("sets node type to stepNode", () => {
    const step = createStep("step1");

    const { result } = renderHook(() => useNodeCalculation([step], []));

    expect(result.current[0].type).toBe("stepNode");
  });

  test("passes executions to node data", () => {
    const step = createStep("step1");
    const executions: any = [{ step_id: "step1", status: "completed" }];

    const { result } = renderHook(() =>
      useNodeCalculation([step], [], undefined, executions)
    );

    expect(result.current[0].data.executions).toEqual(executions);
  });

  test("passes resolved attributes to node data", () => {
    const step = createStep("step1");
    const resolvedAttributes = ["attr1", "attr2"];

    const { result } = renderHook(() =>
      useNodeCalculation(
        [step],
        [],
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        resolvedAttributes
      )
    );

    expect(result.current[0].data.resolvedAttributes).toEqual(
      resolvedAttributes
    );
  });

  test("passes disableEdit flag to node data", () => {
    const step = createStep("step1");

    const { result } = renderHook(() =>
      useNodeCalculation(
        [step],
        [],
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        true
      )
    );

    expect(result.current[0].data.disableEdit).toBe(true);
  });
});
