import React from "react";
import { useNavigate } from "react-router-dom";
import { Step } from "@/app/api";
import { isStepEditable } from "@/app/components/organisms/OverviewStep/useWidget";
import { useUI } from "@/app/contexts/UIContext";
import { useStepEditorContext } from "@/app/contexts/StepEditorContext";
import { useChatAction } from "@/app/hooks/useChatAction";
import { useSpaces, useSteps, useUpsertStep } from "@/app/store/flowStore";

export const useOverviewDiagram = (
  openEditor: ReturnType<typeof useStepEditorContext>["openEditor"]
) => {
  const steps = useSteps();
  const spaces = useSpaces();
  const applyStepUpdate = useUpsertStep();
  const navigate = useNavigate();
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, setGoalSteps, setSpaceId, panelRef } = useUI();

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

  useChatAction({
    open_flow: ({ flow_id }) => navigate(`/flow/${flow_id}`),
    open_overview: ({ space_id }) => {
      const space = spaces.find((candidate) => candidate.id === space_id);
      if (space) setSpaceId(space.id);
    },
    open_step: ({ step_id }) => {
      const step = steps.find((candidate) => candidate.id === step_id);
      if (!step || !isStepEditable(step)) return;
      openEditor({ step, diagramContainerRef, onUpdate: applyStepUpdate });
    },
  });

  return {
    steps,
    diagramContainerRef,
    panelRef,
    selection: { goalSteps, setGoalSteps },
    handleCreateStep,
  };
};
