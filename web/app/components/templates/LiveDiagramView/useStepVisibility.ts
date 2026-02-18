import { Step, FlowContext } from "@/app/api";
import { getStepsFromPlan } from "@/utils/planUtils";

export interface StepVisibilityResult {
  visibleSteps: Step[];
}

export function useStepVisibility(
  steps: Step[] = [],
  flowData?: FlowContext | null
): StepVisibilityResult {
  if (flowData?.plan?.steps && Object.keys(flowData.plan.steps).length > 0) {
    return {
      visibleSteps: getStepsFromPlan(flowData.plan),
    };
  }

  return {
    visibleSteps: steps,
  };
}
