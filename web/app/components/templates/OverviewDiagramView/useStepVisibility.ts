import { useMemo } from "react";
import { Step, ExecutionPlan } from "@/app/api";
import { getStepsFromPlan } from "@/utils/planUtils";

export interface StepVisibilityResult {
  visibleSteps: Step[];
  previewStepIds: Set<string> | null;
}

export function useStepVisibility(
  steps: Step[] = [],
  previewPlan?: ExecutionPlan | null,
  selected?: Set<string> | null
): StepVisibilityResult {
  return useMemo(() => {
    const scopedSteps = selected
      ? steps.filter((step) => selected.has(step.id))
      : steps;

    if (previewPlan?.steps) {
      const planSteps = getStepsFromPlan(previewPlan);
      const planStepIds = new Set(planSteps.map((step) => step.id));
      return {
        visibleSteps: scopedSteps,
        previewStepIds: planStepIds,
      };
    }

    return {
      visibleSteps: scopedSteps,
      previewStepIds: null,
    };
  }, [steps, previewPlan, selected]);
}
