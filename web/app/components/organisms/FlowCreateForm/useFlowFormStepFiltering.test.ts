import { renderHook } from "@testing-library/react";
import { AttributeRole, AttributeType, ExecutionPlan, Step } from "@/app/api";
import { useFlowFormStepFiltering } from "./useFlowFormStepFiltering";

const buildStep = (id: string, attributes: Step["attributes"]): Step => ({
  id,
  name: id,
  type: "sync",
  attributes,
});

describe("useFlowFormStepFiltering", () => {
  const steps: Step[] = [
    buildStep("step-1", {
      input: { role: AttributeRole.Required, type: AttributeType.String },
      outputA: { role: AttributeRole.Output, type: AttributeType.String },
    }),
    buildStep("step-2", {
      outputB: { role: AttributeRole.Output, type: AttributeType.String },
      outputC: { role: AttributeRole.Output, type: AttributeType.String },
    }),
    buildStep("step-3", {
      optional: { role: AttributeRole.Optional, type: AttributeType.Number },
    }),
  ];

  const previewPlan: ExecutionPlan = {
    goals: [],
    required: [],
    steps: {
      "step-1": steps[0],
      "step-2": steps[1],
    },
    attributes: {},
  };

  it("builds included/satisfied sets from plan", () => {
    const initialState = '{"outputA":"value","outputB":"value","extra":"data"}';
    const { result } = renderHook(() =>
      useFlowFormStepFiltering(steps, initialState, previewPlan)
    );

    expect(result.current.parsedState).toEqual({
      outputA: "value",
      outputB: "value",
      extra: "data",
    });
    expect(result.current.included.has("step-1")).toBe(true);
    expect(result.current.included.has("step-2")).toBe(true);

    expect(result.current.satisfied.has("step-1")).toBe(true);
    expect(result.current.satisfied.has("step-2")).toBe(false);
    expect(result.current.satisfied.has("step-3")).toBe(false);
  });

  it("uses excluded steps when available", () => {
    const resolvedPlan: ExecutionPlan = {
      goals: [],
      required: [],
      steps: {
        "step-1": steps[0],
      },
      attributes: {},
      excluded: {
        satisfied: {
          "step-2": ["outputB"],
        },
        missing: {
          "step-3": ["optional"],
        },
      },
    };

    const { result } = renderHook(() =>
      useFlowFormStepFiltering(steps, "{}", resolvedPlan)
    );

    expect(result.current.satisfied.has("step-2")).toBe(true);
    expect(result.current.missingByStep.get("step-3")).toEqual(["optional"]);
  });

  it("returns empty sets without preview plan", () => {
    const { result } = renderHook(() =>
      useFlowFormStepFiltering(steps, "{}", null)
    );

    expect(result.current.included.size).toBe(0);
    expect(result.current.satisfied.size).toBe(0);
    expect(result.current.missingByStep.size).toBe(0);
  });

  it("falls back on invalid initial state", () => {
    const { result } = renderHook(() =>
      useFlowFormStepFiltering(steps, "{invalid", previewPlan)
    );

    expect(result.current.parsedState).toEqual({});
    expect(result.current.satisfied.size).toBe(0);
    expect(result.current.missingByStep.size).toBe(0);
  });
});
