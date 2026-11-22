import React, { useCallback } from "react";
import { useDiagramSelection } from "@/app/contexts/DiagramSelectionContext";

export const useNodeClick = (
  stepId: string,
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void
) => {
  const { setGoalSteps } = useDiagramSelection();

  // Memoize the click handler to prevent unnecessary re-renders
  return useCallback(
    (event: React.MouseEvent) => {
      const additive = event.ctrlKey || event.metaKey;
      if (onStepClick) {
        onStepClick(stepId, { additive });
      } else if (!additive) {
        setGoalSteps([[stepId]]);
      }
    },
    [onStepClick, setGoalSteps, stepId]
  );
};
