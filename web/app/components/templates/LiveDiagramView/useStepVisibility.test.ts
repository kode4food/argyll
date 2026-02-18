import { renderHook } from "@testing-library/react";
import { useStepVisibility } from "./useStepVisibility";
import type { Step, FlowContext } from "@/app/api";
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

  test("returns flow plan steps when available", () => {
    const step1 = createStep("step1");
    const step2 = createStep("step2");
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
      plan: {
        goals: ["step1"],
        required: [],
        steps: {
          step1: step1,
        },
        attributes: {},
      },
    };

    const { result } = renderHook(() =>
      useStepVisibility([step1, step2], flowData)
    );

    expect(result.current.visibleSteps).toEqual([step1]);
  });

  test("falls back to engine steps when no plan", () => {
    const step1 = createStep("step1");
    const step2 = createStep("step2");
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useStepVisibility([step1, step2], flowData)
    );

    expect(result.current.visibleSteps).toEqual([step1, step2]);
  });

  test("updates visible steps when plan changes for same flow", () => {
    const step1 = createStep("step1");
    const step2 = createStep("step2");
    const baseFlow: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const initialFlow = {
      ...baseFlow,
      plan: {
        goals: ["step1"],
        required: [],
        steps: {
          step1,
        },
        attributes: {},
      },
    };

    const updatedFlow = {
      ...baseFlow,
      plan: {
        goals: ["step2"],
        required: [],
        steps: {
          step1,
          step2,
        },
        attributes: {},
      },
    };

    const { result, rerender } = renderHook(
      ({ flowData }) => useStepVisibility([step1, step2], flowData),
      {
        initialProps: { flowData: initialFlow },
      }
    );

    expect(result.current.visibleSteps).toEqual([step1]);

    rerender({ flowData: updatedFlow });

    expect(result.current.visibleSteps).toEqual([step1, step2]);
  });
});
