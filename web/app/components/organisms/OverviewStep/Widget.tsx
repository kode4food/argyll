import React from "react";
import { Step } from "@/app/api";
import StepHeader from "@/app/components/molecules/StepHeader";
import Attributes from "@/app/components/molecules/OverviewStep/Attributes";
import StepPredicate from "@/app/components/molecules/StepPredicate";
import Footer from "@/app/components/molecules/OverviewStep/Footer";
import { useT } from "@/app/i18n";
import { useWidget } from "./useWidget";

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
}) => {
  const t = useT();
  const {
    healthStatus,
    healthError,
    healthNodes,
    stepType,
    isGrayedOut,
    isEditable,
    focusedAttributeName,
    handleDoubleClick,
  } = useWidget({ step, isInPreviewPlan, isPreviewMode, diagramContainerRef });

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
        <Attributes step={step} focusedAttributeName={focusedAttributeName} />
        <StepPredicate step={step} />
        <Footer
          step={step}
          healthStatus={healthStatus}
          healthError={healthError}
          healthNodes={healthNodes}
        />
      </div>
    </>
  );
};

export default React.memo(Widget);
