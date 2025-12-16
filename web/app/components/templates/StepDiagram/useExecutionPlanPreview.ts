import { useCallback, useEffect } from "react";
import { ExecutionPlan, FlowContext } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";

export interface UseExecutionPlanPreviewReturn {
  previewPlan: ExecutionPlan | null;
  handleStepClick: (
    stepId: string,
    options?: { additive?: boolean }
  ) => Promise<void>;
  clearPreview: () => void;
}

export function useExecutionPlanPreview(
  goalSteps: string[],
  setGoalSteps: (stepIds: string[]) => void,
  flowData?: FlowContext | null
): UseExecutionPlanPreviewReturn {
  const { previewPlan, updatePreviewPlan, clearPreviewPlan } = useUI();

  const handleStepClick = useCallback(
    async (stepId: string, options?: { additive?: boolean }) => {
      if (flowData) {
        return;
      }

      const isAdditive = options?.additive ?? false;

      if (isAdditive) {
        const isIncludedByPlan =
          !!previewPlan?.steps?.[stepId] && !goalSteps.includes(stepId);
        if (isIncludedByPlan) {
          return;
        }

        const nextGoals = goalSteps.includes(stepId)
          ? goalSteps.filter((id) => id !== stepId)
          : [...goalSteps, stepId];

        setGoalSteps(nextGoals);
        if (nextGoals.length === 0) {
          clearPreviewPlan();
        } else {
          await updatePreviewPlan(nextGoals, {});
        }
        return;
      }

      const isCurrentlySingleSelection =
        goalSteps.length === 1 && goalSteps[0] === stepId;

      if (isCurrentlySingleSelection) {
        setGoalSteps([]);
        clearPreviewPlan();
        return;
      }

      const nextGoals = [stepId];
      setGoalSteps(nextGoals);
      await updatePreviewPlan(nextGoals, {});
    },
    [
      flowData,
      goalSteps,
      setGoalSteps,
      updatePreviewPlan,
      clearPreviewPlan,
      previewPlan,
    ]
  );

  const clearPreview = useCallback(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [setGoalSteps, clearPreviewPlan]);

  useEffect(() => {
    if (flowData) {
      clearPreviewPlan();
    }
  }, [flowData, clearPreviewPlan]);

  return {
    previewPlan,
    handleStepClick,
    clearPreview,
  };
}
