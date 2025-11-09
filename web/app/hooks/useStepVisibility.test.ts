import { renderHook } from "@testing-library/react";
import { useStepVisibility } from "./useStepVisibility";
import type { Step, WorkflowContext, ExecutionPlan } from "../api";
import { AttributeRole, AttributeType } from "../api";

describe("useStepVisibility", () => {
  const createStep = (id: string, outputs: string[]): Step => ({
    id,
    name: `Step ${id}`,
    type: "sync",
    attributes: Object.fromEntries(
      outputs.map((name) => [
        name,
        { role: AttributeRole.Output, type: AttributeType.String },
      ])
    ),
    version: "1.0.0",
    http: {
      endpoint: "http://test",
      timeout: 5000,
    },
  });

  test("returns all steps when no workflow or preview", () => {
    const steps = [
      createStep("step1", ["out1"]),
      createStep("step2", ["out2"]),
    ];
    const { result } = renderHook(() => useStepVisibility(steps));

    expect(result.current.visibleSteps).toEqual(steps);
    expect(result.current.previewStepIds).toBeNull();
  });

  test("filters steps by workflow execution plan", () => {
    const step1 = createStep("step1", ["out1"]);
    const step2 = createStep("step2", ["out2"]);
    const steps = [step1, step2];

    const workflowData: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
      plan: {
        goals: ["step1"],
        required: [],
        steps: {
          step1: { step: step1 },
        },
        attributes: {},
      },
    };

    const { result } = renderHook(() =>
      useStepVisibility(steps, workflowData, null)
    );

    expect(result.current.visibleSteps).toEqual([step1]);
    expect(result.current.previewStepIds).toBeNull();
  });

  test("returns all steps with preview IDs for preview plan", () => {
    const step1 = createStep("step1", ["out1"]);
    const step2 = createStep("step2", ["out2"]);
    const steps = [step1, step2];

    const previewPlan: ExecutionPlan = {
      goals: ["step1"],
      required: [],
      steps: {
        step1: { step: step1 },
      },
      attributes: {},
    };

    const { result } = renderHook(() =>
      useStepVisibility(steps, null, previewPlan)
    );

    expect(result.current.visibleSteps).toEqual(steps);
    expect(result.current.previewStepIds?.has("step1")).toBe(true);
    expect(result.current.previewStepIds?.has("step2")).toBe(false);
  });

  test("workflow plan takes precedence over preview plan", () => {
    const step1 = createStep("step1", ["out1"]);
    const step2 = createStep("step2", ["out2"]);
    const steps = [step1, step2];

    const workflowData: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
      plan: {
        goals: ["step1"],
        required: [],
        steps: {
          step1: { step: step1 },
        },
        attributes: {},
      },
    };

    const previewPlan: ExecutionPlan = {
      goals: ["step2"],
      required: [],
      steps: {
        step2: { step: step2 },
      },
      attributes: {},
    };

    const { result } = renderHook(() =>
      useStepVisibility(steps, workflowData, previewPlan)
    );

    expect(result.current.visibleSteps).toEqual([step1]);
    expect(result.current.previewStepIds).toBeNull();
  });

  test("handles empty steps array", () => {
    const { result } = renderHook(() => useStepVisibility([]));

    expect(result.current.visibleSteps).toEqual([]);
    expect(result.current.previewStepIds).toBeNull();
  });
});
