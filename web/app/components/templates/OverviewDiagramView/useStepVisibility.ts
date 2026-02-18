import { useMemo } from "react";
import { Step, ExecutionPlan } from "@/app/api";
import { getStepsFromPlan } from "@/utils/planUtils";

export interface StepVisibilityResult {
  visibleSteps: Step[];
  previewStepIds: Set<string> | null;
}

export function useStepVisibility(
  steps: Step[] = [],
  previewPlan?: ExecutionPlan | null
): StepVisibilityResult {
  return useMemo(() => {
    if (previewPlan?.steps) {
      const planSteps = getStepsFromPlan(previewPlan);
      const planStepIds = new Set(planSteps.map((step) => step.id));
      return {
        visibleSteps: steps,
        previewStepIds: planStepIds,
      };
    }

    return {
      visibleSteps: steps,
      previewStepIds: null,
    };
  }, [steps, previewPlan]);
}
