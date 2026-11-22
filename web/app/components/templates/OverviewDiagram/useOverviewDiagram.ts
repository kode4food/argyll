import React from "react";
import { Step } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import { useStepEditorContext } from "@/app/contexts/StepEditorContext";
import { useSteps, useUpsertStep } from "@/app/store/flowStore";

export const useOverviewDiagram = (
  openEditor: ReturnType<typeof useStepEditorContext>["openEditor"]
) => {
  const steps = useSteps();
  const applyStepUpdate = useUpsertStep();
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, setGoalSteps, panelRef } = useUI();

  const handleCreateStep = React.useCallback(
    (draft?: Step) => {
      openEditor({
        step: null,
        diagramContainerRef,
        onUpdate: applyStepUpdate,
        draft,
      });
    },
    [applyStepUpdate, openEditor]
  );

  return {
    steps,
    diagramContainerRef,
    panelRef,
    selection: { goalSteps, setGoalSteps },
    handleCreateStep,
  };
};
