import React, { useState } from "react";
import { ExecutionResult, Step } from "../../api";
import StepHeader from "../molecules/StepHeader";
import StepAttributesSection from "../molecules/StepAttributesSection";
import StepPredicate from "../molecules/StepPredicate";
import StepFooter from "../molecules/StepFooter";
import { getStepType } from "@/utils/stepUtils";
import { useStepHealth } from "../../hooks/useStepHealth";
import { useStepEditorContext } from "../../contexts/StepEditorContext";
import { useFlowSession } from "../../contexts/FlowSessionContext";

interface StepWidgetProps {
  step: Step;
  selected?: boolean;
  onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
  mode?: "list" | "diagram";
  style?: React.CSSProperties;
  className?: string;
  execution?: ExecutionResult;
  satisfiedArgs?: Set<string>;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  flowId?: string;
  attributeProvenance?: Map<string, string>;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  disableEdit?: boolean;
}

const StepWidget: React.FC<StepWidgetProps> = ({
  step,
  selected = false,
  onClick,
  mode = "list",
  style,
  className = "",
  execution,
  satisfiedArgs = new Set(),
  isInPreviewPlan = true,
  isPreviewMode = false,
  flowId,
  attributeProvenance = new Map(),
  diagramContainerRef,
  disableEdit = false,
}) => {
  const { status: healthStatus, error: healthError } = useStepHealth(step);
  const stepType = getStepType(step);

  const [localStep, setLocalStep] = useState(step);
  const { openEditor } = useStepEditorContext();
  const { loadSteps } = useFlowSession();

  React.useEffect(() => {
    const handleOpenEditor = (event: Event) => {
      const customEvent = event as CustomEvent;
      if (customEvent.detail?.stepId === step.id && !disableEdit) {
        openEditor({
          step: localStep,
          onUpdate: async (updated) => {
            setLocalStep(updated);
            await loadSteps();
          },
          diagramContainerRef,
        });
      }
    };

    document.addEventListener("openStepEditor", handleOpenEditor);
    return () =>
      document.removeEventListener("openStepEditor", handleOpenEditor);
  }, [
    step.id,
    disableEdit,
    openEditor,
    localStep,
    diagramContainerRef,
    loadSteps,
  ]);

  const isGrayedOut = isPreviewMode && !isInPreviewPlan;
  const isEditable =
    !disableEdit &&
    !flowId &&
    ((localStep.type === "script" && localStep.script) ||
      ((localStep.type === "sync" || localStep.type === "async") &&
        localStep.http));

  const handleDoubleClick = (e: React.MouseEvent) => {
    if (disableEdit || !isEditable) return;
    e.stopPropagation();
    openEditor({
      step: localStep,
      onUpdate: async (updated) => {
        setLocalStep(updated);
        await loadSteps();
      },
      diagramContainerRef,
    });
  };

  const widgetClassName = [
    "step-widget",
    stepType,
    mode,
    selected && "selected",
    onClick && "clickable",
    isGrayedOut && "grayed-out",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <>
      <div
        className={widgetClassName}
        style={style}
        onClick={onClick}
        onDoubleClick={handleDoubleClick}
        title={isEditable ? "Double-click to edit step" : undefined}
      >
        <StepHeader step={step} />
        <StepAttributesSection
          step={step}
          satisfiedArgs={satisfiedArgs}
          showStatus={execution !== undefined || flowId !== undefined}
          execution={execution}
          attributeProvenance={attributeProvenance}
        />
        <StepPredicate step={step} />
        <StepFooter
          step={step}
          healthStatus={healthStatus}
          healthError={healthError}
          flowId={flowId}
          execution={execution}
        />
      </div>
    </>
  );
};

export default React.memo(StepWidget);
