import React, { useCallback } from "react";
import { Step } from "@/app/api";
import StepHeader from "@/app/components/molecules/StepHeader";
import Attributes from "@/app/components/molecules/OverviewStep/Attributes";
import StepPredicate from "@/app/components/molecules/StepPredicate";
import Footer from "@/app/components/molecules/OverviewStep/Footer";
import { getStepType } from "@/utils/stepUtils";
import { useStepHealth } from "@/app/hooks/useStepHealth";
import { useStepEditorContext } from "@/app/contexts/StepEditorContext";
import { useT } from "@/app/i18n";
import { useFlowStore } from "@/app/store/flowStore";

interface WidgetProps {
  step: Step;
  selected?: boolean;
  onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
  mode?: "list" | "diagram";
  style?: React.CSSProperties;
  className?: string;
  isInPreviewPlan?: boolean;
  isPreviewMode?: boolean;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  disableEdit?: boolean;
}

const Widget: React.FC<WidgetProps> = ({
  step,
  selected = false,
  onClick,
  mode = "list",
  style,
  className = "",
  isInPreviewPlan = true,
  isPreviewMode = false,
  diagramContainerRef,
  disableEdit = false,
}) => {
  const { status: healthStatus, error: healthError } = useStepHealth(step);
  const stepType = getStepType(step);
  const addStep = useFlowStore((state) => state.addStep);
  const updateStep = useFlowStore((state) => state.updateStep);
  const { openEditor } = useStepEditorContext();
  const t = useT();

  const handleStepUpdate = useCallback(
    (updatedStep: Step) => {
      const { steps } = useFlowStore.getState();
      const hasExisting = steps.some((currentStep) => {
        return currentStep.id === updatedStep.id;
      });
      if (hasExisting) {
        updateStep(updatedStep);
        return;
      }
      addStep(updatedStep);
    },
    [addStep, updateStep]
  );

  React.useEffect(() => {
    const handleOpenEditor = (event: Event) => {
      const customEvent = event as CustomEvent;
      if (customEvent.detail?.stepId === step.id && !disableEdit) {
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
  }, [
    step,
    step.id,
    disableEdit,
    openEditor,
    handleStepUpdate,
    diagramContainerRef,
  ]);

  const isGrayedOut = isPreviewMode && !isInPreviewPlan;
  const isEditable =
    !disableEdit &&
    ((step.type === "script" && step.script) ||
      ((step.type === "sync" || step.type === "async") && step.http) ||
      (step.type === "flow" && step.flow));

  const handleDoubleClick = (e: React.MouseEvent) => {
    if (disableEdit || !isEditable) return;
    e.stopPropagation();
    openEditor({
      step,
      onUpdate: handleStepUpdate,
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
        title={isEditable ? t("overviewStep.doubleClickEdit") : undefined}
      >
        <StepHeader step={step} />
        <Attributes step={step} />
        <StepPredicate step={step} />
        <Footer
          step={step}
          healthStatus={healthStatus}
          healthError={healthError}
        />
      </div>
    </>
  );
};

export default React.memo(Widget);
