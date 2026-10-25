import { useCallback } from "react";
import { ExecutionPlan, GoalSets, Step } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import {
  activeGoalSet,
  nonEmptyGoalSets,
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
}

export function useExecutionPlanPreview(
  steps: Step[],
  goalSteps: GoalSets,
  setGoalSteps: (goals: GoalSets) => void
): UseExecutionPlanPreviewReturn {
  const { previewPlan, updatePreviewPlan, clearPreviewPlan } = useUI();

  // Preview focus never changes the full plan used to create the Flow
  const applyActiveSet = useCallback(
    async (set: string[]) => {
      const nextGoals = withActiveGoalSet(goalSteps, set);
      setGoalSteps(nextGoals);
      if (nonEmptyGoalSets(nextGoals).length === 0) {
        clearPreviewPlan();
        return;
      }
      await updatePreviewPlan(nextGoals, {});
    },
    [goalSteps, setGoalSteps, updatePreviewPlan, clearPreviewPlan]
  );

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

        await applyActiveSet(toggleGoal(steps, active, stepId));
        return;
      }

      const isCurrentlySingleSelection =
        active.length === 1 && active[0] === stepId;
      await applyActiveSet(isCurrentlySingleSelection ? [] : [stepId]);
    },
    [goalSteps, applyActiveSet, steps]
  );

  return {
    previewPlan,
    handleStepClick,
  };
}
