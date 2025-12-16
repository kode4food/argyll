import { useMemo } from "react";
import { Step, FlowContext, ExecutionPlan } from "@/app/api";
import { getStepsFromPlan } from "@/utils/planUtils";

export interface StepVisibilityResult {
  visibleSteps: Step[];
  previewStepIds: Set<string> | null;
}

export function useStepVisibility(
  steps: Step[] = [],
  flowData?: FlowContext | null,
  previewPlan?: ExecutionPlan | null
): StepVisibilityResult {
  return useMemo(() => {
    if (flowData?.plan?.steps) {
      const planSteps = getStepsFromPlan(flowData.plan);
      const planStepIds = new Set(planSteps.map((step) => step.id));
      return {
        visibleSteps: (steps || []).filter((step) => planStepIds.has(step.id)),
        previewStepIds: null,
      };
    }

    if (previewPlan?.steps) {
      const planSteps = getStepsFromPlan(previewPlan);
      const planStepIds = new Set(planSteps.map((step) => step.id));
      return {
        visibleSteps: steps || [],
        previewStepIds: planStepIds,
      };
    }

    return {
      visibleSteps: steps || [],
      previewStepIds: null,
    };
  }, [steps, flowData, previewPlan]);
}
