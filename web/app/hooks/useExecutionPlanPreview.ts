import { useCallback, useEffect } from "react";
import { ExecutionPlan, WorkflowContext } from "../api";
import { useUI } from "../contexts/UIContext";

export interface UseExecutionPlanPreviewReturn {
  previewPlan: ExecutionPlan | null;
  handleStepClick: (stepId: string) => Promise<void>;
  clearPreview: () => void;
}

export function useExecutionPlanPreview(
  selectedStep: string | null,
  onSelectStep: (stepId: string | null) => void,
  workflowData?: WorkflowContext | null
): UseExecutionPlanPreviewReturn {
  const { previewPlan, updatePreviewPlan, clearPreviewPlan } = useUI();

  const handleStepClick = useCallback(
    async (stepId: string) => {
      if (workflowData) {
        return;
      }

      if (selectedStep === stepId) {
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
      workflowData,
      selectedStep,
      onSelectStep,
      updatePreviewPlan,
      clearPreviewPlan,
    ]
  );

  const clearPreview = useCallback(() => {
    clearPreviewPlan();
    onSelectStep(null);
  }, [onSelectStep, clearPreviewPlan]);

  useEffect(() => {
    if (workflowData) {
      clearPreviewPlan();
    }
  }, [workflowData, clearPreviewPlan]);

  return {
    previewPlan,
    handleStepClick,
    clearPreview,
  };
}
