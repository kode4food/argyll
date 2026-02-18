import { useCallback } from "react";
import { Step } from "@/app/api";

export function useStepEditorIntegration(
  openStepEditor: (step: Step | null) => void,
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

  const handleOpenEditor = useCallback(
    (step: Step | null) => {
      openStepEditor(step);
    },
    [openStepEditor]
  );

  return {
    handleStepCreated,
    openStepEditor: handleOpenEditor,
  };
}
