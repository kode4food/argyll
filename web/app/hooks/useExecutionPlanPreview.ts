import { useCallback, useEffect } from "react";
import { ExecutionPlan, FlowContext } from "../api";
import { useUI } from "../contexts/UIContext";

export interface UseExecutionPlanPreviewReturn {
  previewPlan: ExecutionPlan | null;
  handleStepClick: (
    stepId: string,
    options?: { additive?: boolean }
  ) => Promise<void>;
  clearPreview: () => void;
}

export function useExecutionPlanPreview(
  goalStepIds: string[],
  onSelectStep: (stepId: string | null) => void,
  onToggleStep: (stepId: string) => void,
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
          !!previewPlan?.steps?.[stepId] && !goalStepIds.includes(stepId);
        if (isIncludedByPlan) {
          return;
        }
        const nextGoals = goalStepIds.includes(stepId)
          ? goalStepIds.filter((id) => id !== stepId)
          : [...goalStepIds, stepId];

        onToggleStep(stepId);
        await updatePreviewPlan(nextGoals, {});
        return;
      }

      const isCurrentlySingleSelection =
        goalStepIds.length === 1 && goalStepIds[0] === stepId;

      if (isCurrentlySingleSelection) {
        // Deselect current step
        onSelectStep(null);
        clearPreviewPlan();
        return;
      }

      // Set selected step immediately (optimistically) for instant visual feedback
      onSelectStep(stepId);

      // Then load the preview plan (async operation)
      // The AbortController in UIContext will handle race conditions
      await updatePreviewPlan([stepId], {});
    },
    [
      flowData,
      goalStepIds,
      onSelectStep,
      onToggleStep,
      updatePreviewPlan,
      clearPreviewPlan,
      previewPlan,
    ]
  );

  const clearPreview = useCallback(() => {
    clearPreviewPlan();
    onSelectStep(null);
  }, [onSelectStep, clearPreviewPlan]);

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
