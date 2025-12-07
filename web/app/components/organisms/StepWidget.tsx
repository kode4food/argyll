import React, { useState, lazy, Suspense } from "react";
import { ExecutionResult, Step } from "../../api";
import StepHeader from "../molecules/StepHeader";
import StepAttributesSection from "../molecules/StepAttributesSection";
import StepPredicate from "../molecules/StepPredicate";
import StepFooter from "../molecules/StepFooter";
import { getStepType } from "@/utils/stepUtils";
import { useStepHealth } from "../../hooks/useStepHealth";

const StepEditor = lazy(() => import("./StepEditor"));

interface StepWidgetProps {
  step: Step;
  selected?: boolean;
  onClick?: () => void;
  mode?: "list" | "diagram";
  style?: React.CSSProperties;
  className?: string;
  execution?: ExecutionResult;
  satisfiedArgs?: Set<string>;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  flowId?: string;
  attributeProvenance?: Map<string, string>;
  diagramContainerRef?:
    | React.RefObject<HTMLDivElement | null>
    | React.MutableRefObject<HTMLDivElement | null>;
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

  const [showEditor, setShowEditor] = useState(false);
  const [localStep, setLocalStep] = useState(step);

  React.useEffect(() => {
    const handleOpenEditor = (event: Event) => {
      const customEvent = event as CustomEvent;
      if (customEvent.detail?.stepId === step.id && !disableEdit) {
        setShowEditor(true);
      }
    };

    document.addEventListener("openStepEditor", handleOpenEditor);
    return () =>
      document.removeEventListener("openStepEditor", handleOpenEditor);
  }, [step.id, disableEdit]);

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
    setShowEditor(true);
  };

  const handleStepUpdate = (updatedStep: Step) => {
    setLocalStep(updatedStep);
  };

  return (
    <>
      <div
        className={`step-widget ${stepType} ${mode} ${selected ? "selected" : ""} ${onClick ? "clickable" : ""} ${isGrayedOut ? "grayed-out" : ""} ${className}`}
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

      {showEditor && (
        <Suspense fallback={null}>
          <StepEditor
            step={localStep}
            onClose={() => setShowEditor(false)}
            onUpdate={handleStepUpdate}
            diagramContainerRef={diagramContainerRef}
          />
        </Suspense>
      )}
    </>
  );
};

export default React.memo(StepWidget);
