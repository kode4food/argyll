import { renderHook } from "@testing-library/react";
import { useNodeCalculation } from "./useNodeCalculation";
import type { Step } from "@/app/api";
import { AttributeRole } from "@/app/api";

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(() => ({})),
}));

describe("useNodeCalculation", () => {
  const createStep = (id: string): Step => ({
    id,
    name: `Step ${id}`,
    type: "service",
    attributes: {
      out: { role: AttributeRole.Output },
    },
    http: {
      endpoint: "http://test",
      timeout: 5000,
    },
  });

  test("creates nodes for visible steps", () => {
    const steps = [createStep("step1"), createStep("step2")];
    const { result } = renderHook(() =>
      useNodeCalculation({ visibleSteps: steps, selectedStepIds: [] })
    );

    expect(result.current).toHaveLength(2);
  });

  test("marks selected nodes", () => {
    const step = createStep("step1");
    const { result } = renderHook(() =>
      useNodeCalculation({ visibleSteps: [step], selectedStepIds: ["step1"] })
    );

    expect(result.current[0].data.selected).toBe(true);
  });

  test("sets preview flags", () => {
    const step = createStep("step1");
    const previewStepIds = new Set(["step1"]);
    const previewPlan: any = {
      goals: { steps: ["step1"] },
      steps: { step1: step },
    };

    const { result } = renderHook(() =>
      useNodeCalculation({
        visibleSteps: [step],
        selectedStepIds: [],
        previewPlan,
        previewStepIds,
        onStepClick: jest.fn(),
      })
    );

    expect(result.current[0].data.isPreviewMode).toBe(true);
    expect(result.current[0].data.isInPreviewPlan).toBe(true);
  });
});
