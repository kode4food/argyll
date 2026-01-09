import { renderHook } from "@testing-library/react";
import { useStepVisibility } from "./useStepVisibility";
import type { Step, ExecutionPlan } from "@/app/api";
import { AttributeRole, AttributeType } from "@/app/api";

describe("useStepVisibility", () => {
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

  test("returns all steps when no preview", () => {
    const steps = [createStep("step1"), createStep("step2")];
    const { result } = renderHook(() => useStepVisibility(steps));

    expect(result.current.visibleSteps).toEqual(steps);
    expect(result.current.previewStepIds).toBeNull();
  });

  test("returns preview ids when preview plan exists", () => {
    const step1 = createStep("step1");
    const step2 = createStep("step2");
    const steps = [step1, step2];

    const previewPlan: ExecutionPlan = {
      goals: ["step1"],
      required: [],
      steps: {
        step1: step1,
      },
      attributes: {},
    };

    const { result } = renderHook(() => useStepVisibility(steps, previewPlan));

    expect(result.current.visibleSteps).toEqual(steps);
    expect(result.current.previewStepIds?.has("step1")).toBe(true);
    expect(result.current.previewStepIds?.has("step2")).toBe(false);
  });
});
