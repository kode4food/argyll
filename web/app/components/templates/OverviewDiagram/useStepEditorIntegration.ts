import { useCallback } from "react";
import { Step } from "@/app/api";

export function useStepEditorIntegration(
  refreshSteps?: () => Promise<void>,
  applyStepUpdate?: (step: Step) => void
) {
  const handleStepCreated = useCallback(
    async (step?: Step) => {
      if (step && applyStepUpdate) {
        applyStepUpdate(step);
        return;
      }

      if (refreshSteps) {
        await refreshSteps();
      }
    },
    [applyStepUpdate, refreshSteps]
  );

  return {
    handleStepCreated,
  };
}
