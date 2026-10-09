import { useCallback } from "react";
import { ExecutionPlan, GoalSets, Step } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import {
  activeGoalSet,
  toggleGoal,
  upstreamSteps,
  withActiveGoalSet,
} from "@/utils/goalSets";

export interface UseExecutionPlanPreviewReturn {
  previewPlan: ExecutionPlan | null;
  handleStepClick: (
    stepId: string,
    options?: { additive?: boolean }
  ) => Promise<void>;
  clearPreview: () => void;
}

export function useExecutionPlanPreview(
  steps: Step[],
  goalSteps: GoalSets,
  setGoalSteps: (goals: GoalSets) => void
): UseExecutionPlanPreviewReturn {
  const { previewPlan, updatePreviewPlan, clearPreviewPlan } = useUI();

  const handleStepClick = useCallback(
    async (stepId: string, options?: { additive?: boolean }) => {
      const isAdditive = options?.additive ?? false;
      const active = activeGoalSet(goalSteps);

      if (isAdditive) {
        // Only the active AND-set's own upstream Steps are already included
        const isIncludedByActiveSet =
          upstreamSteps(steps, active).has(stepId) && !active.includes(stepId);
        if (isIncludedByActiveSet) {
          return;
        }

        const nextGoals = withActiveGoalSet(
          goalSteps,
          toggleGoal(steps, active, stepId)
        );

        setGoalSteps(nextGoals);
        await updatePreviewPlan(nextGoals, {});
        return;
      }

      const isCurrentlySingleSelection =
        goalSteps.length === 1 && active.length === 1 && active[0] === stepId;

      if (isCurrentlySingleSelection) {
        setGoalSteps([]);
        clearPreviewPlan();
        return;
      }

      const nextGoals = [[stepId]];
      setGoalSteps(nextGoals);
      await updatePreviewPlan(nextGoals, {});
    },
    [goalSteps, setGoalSteps, updatePreviewPlan, clearPreviewPlan, steps]
  );

  const clearPreview = useCallback(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [setGoalSteps, clearPreviewPlan]);

  return {
    previewPlan,
    handleStepClick,
    clearPreview,
  };
}
