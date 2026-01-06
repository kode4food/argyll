import { useMemo } from "react";
import { Step, AttributeRole, ExecutionPlan } from "@/app/api";
import { safeParseState } from "./flowFormUtils";

export function useFlowFormStepFiltering(
  steps: Step[],
  initialState: string,
  previewPlan: ExecutionPlan | null
) {
  const included = useMemo(() => {
    if (!previewPlan?.steps) return new Set<string>();
    return new Set(Object.keys(previewPlan.steps));
  }, [previewPlan?.steps]);

  const parsedState = useMemo(
    () => safeParseState(initialState),
    [initialState]
  );

  const satisfied = useMemo(() => {
    const result = new Set<string>();
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
          result.add(step.id);
        }
      }
    });

    return result;
  }, [parsedState, steps]);

  return { included, satisfied, parsedState };
}
