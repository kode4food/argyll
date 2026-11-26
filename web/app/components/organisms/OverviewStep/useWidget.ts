import React from "react";
import { Step } from "@/app/api";
import { useStepHealth } from "@/app/hooks/useStepHealth";
import { useStepEditorContext } from "@/app/contexts/StepEditorContext";
import { useUI } from "@/app/contexts/UIContext";
import { useUpsertStep } from "@/app/store/flowStore";
import { getStepType } from "@/utils/stepUtils";

export const isStepEditable = (step: Step): boolean => {
  switch (step.type) {
    case "script":
      return !!step.script;
    case "service":
      return !!step.http;
    case "flow":
      return !!step.flow;
    case "agent":
      return !!step.agent;
    default:
      return false;
  }
};

export interface WidgetOptions {
  step: Step;
  isInPreviewPlan: boolean;
  isPreviewMode: boolean;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
}

export const useWidget = ({
  step,
  isInPreviewPlan,
  isPreviewMode,
  diagramContainerRef,
}: WidgetOptions) => {
  const {
    status: healthStatus,
    error: healthError,
    nodes: healthNodes,
  } = useStepHealth(step);
  const stepType = getStepType(step);
  const { focusedPreviewAttribute } = useUI();
  const handleStepUpdate = useUpsertStep();
  const { openEditor } = useStepEditorContext();

  React.useEffect(() => {
    const handleOpenEditor = (event: Event) => {
      const customEvent = event as CustomEvent;
      if (customEvent.detail?.stepId === step.id) {
        openEditor({
          step,
          onUpdate: handleStepUpdate,
          diagramContainerRef,
        });
      }
    };

    document.addEventListener("openStepEditor", handleOpenEditor);
    return () =>
      document.removeEventListener("openStepEditor", handleOpenEditor);
  }, [step, step.id, openEditor, handleStepUpdate, diagramContainerRef]);

  const isGrayedOut = isPreviewMode && !isInPreviewPlan;
  const isEditable = isStepEditable(step);
  const focusedAttributeName = isInPreviewPlan ? focusedPreviewAttribute : null;

  const handleDoubleClick = (e: React.MouseEvent) => {
    if (!isEditable) return;
    e.stopPropagation();
    openEditor({
      step,
      onUpdate: handleStepUpdate,
      diagramContainerRef,
    });
  };

  return {
    healthStatus,
    healthError,
    healthNodes,
    stepType,
    isGrayedOut,
    isEditable,
    focusedAttributeName,
    handleDoubleClick,
  };
};
