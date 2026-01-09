import { useCallback } from "react";
import { Step } from "@/app/api";

export function useStepEditorIntegration(
  openStepEditor: (step: Step | null) => void,
  refreshSteps?: () => Promise<void>
) {
  const handleStepCreated = useCallback(async () => {
    if (refreshSteps) {
      await refreshSteps();
    }
  }, [refreshSteps]);

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
