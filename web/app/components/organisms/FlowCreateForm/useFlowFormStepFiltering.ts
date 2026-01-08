import { useMemo } from "react";
import { Step, AttributeRole, ExecutionPlan } from "@/app/api";
import { safeParseState } from "./flowFormUtils";

export function useFlowFormStepFiltering(
  steps: Step[],
  initialState: string,
  previewPlan: ExecutionPlan | null
) {
  const excluded = previewPlan?.excluded;
  const included = useMemo(() => {
    if (!previewPlan?.steps) return new Set<string>();
    return new Set(Object.keys(previewPlan.steps));
  }, [previewPlan?.steps]);

  const parsedState = useMemo(
    () => safeParseState(initialState),
    [initialState]
  );

  const { satisfied, missingByStep } = useMemo(() => {
    const missing = new Map<string, string[]>();
    if (excluded) {
      const satisfiedSteps = new Set<string>();
      const satisfiedMap = excluded.satisfied || {};
      Object.keys(satisfiedMap).forEach((stepId) => {
        satisfiedSteps.add(stepId);
      });
      const missingMap = excluded.missing || {};
      Object.entries(missingMap).forEach(([stepId, names]) => {
        missing.set(stepId, names);
      });
      return { satisfied: satisfiedSteps, missingByStep: missing };
    }

    const fallbackSatisfied = new Set<string>();
    const availableAttrs = new Set(Object.keys(parsedState));

    steps.forEach((step) => {
      const outputKeys = Object.entries(step.attributes || {})
        .filter(([_, spec]) => spec.role === AttributeRole.Output)
        .map(([name]) => name);

      if (outputKeys.length > 0) {
        const allOutputsAvailable = outputKeys.every((name) =>
          availableAttrs.has(name)
        );
        if (allOutputsAvailable) {
          fallbackSatisfied.add(step.id);
        }
      }
    });

    return { satisfied: fallbackSatisfied, missingByStep: missing };
  }, [parsedState, excluded, steps]);

  return { included, satisfied, missingByStep, parsedState };
}
