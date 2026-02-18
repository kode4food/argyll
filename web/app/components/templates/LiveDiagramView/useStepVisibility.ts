import { useMemo, useRef, useEffect } from "react";
import { Step, FlowContext } from "@/app/api";
import { getStepsFromPlan } from "@/utils/planUtils";

export interface StepVisibilityResult {
  visibleSteps: Step[];
}

export function useStepVisibility(
  steps: Step[] = [],
  flowData?: FlowContext | null
): StepVisibilityResult {
  const flowId = flowData?.id ?? null;
  const planStepsRef = useRef<Step[] | null>(null);

  useEffect(() => {
    planStepsRef.current = null;
  }, [flowId]);

  const planSteps = useMemo(() => {
    if (flowData?.plan?.steps && Object.keys(flowData.plan.steps).length > 0) {
      return getStepsFromPlan(flowData.plan);
    }
    return null;
  }, [flowData]);

  if (planSteps) {
    planStepsRef.current = planSteps;
  }

  return {
    visibleSteps: planStepsRef.current || steps,
  };
}
