import { useMemo } from "react";
import { Step, WorkflowContext, ExecutionPlan } from "../api";

export interface StepVisibilityResult {
  visibleSteps: Step[];
  previewStepIds: Set<string> | null;
}

export function useStepVisibility(
  steps: Step[] = [],
  workflowData?: WorkflowContext | null,
  previewPlan?: ExecutionPlan | null
): StepVisibilityResult {
  return useMemo(() => {
    if (workflowData?.execution_plan?.steps) {
      const planStepIds = new Set(
        workflowData.execution_plan.steps.map((step) => step.id)
      );
      return {
        visibleSteps: (steps || []).filter((step) => planStepIds.has(step.id)),
        previewStepIds: null,
      };
    }

    if (previewPlan?.steps) {
      const planStepIds = new Set(previewPlan.steps.map((step) => step.id));
      return {
        visibleSteps: steps || [],
        previewStepIds: planStepIds,
      };
    }

    return {
      visibleSteps: steps || [],
      previewStepIds: null,
    };
  }, [steps, workflowData, previewPlan]);
}
