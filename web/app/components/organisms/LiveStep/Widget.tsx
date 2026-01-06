import React from "react";
import { AttributeValue, ExecutionResult, Step } from "@/app/api";
import StepHeader from "@/app/components/molecules/StepHeader";
import Attributes from "@/app/components/molecules/LiveStep/Attributes";
import StepPredicate from "@/app/components/molecules/StepPredicate";
import Footer from "@/app/components/molecules/LiveStep/Footer";
import { getStepType } from "@/utils/stepUtils";

interface WidgetProps {
  step: Step;
  selected?: boolean;
  onClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
  mode?: "list" | "diagram";
  style?: React.CSSProperties;
  className?: string;
  execution?: ExecutionResult;
  satisfiedArgs?: Set<string>;
  attributeProvenance?: Map<string, string>;
  attributeValues?: Record<string, AttributeValue>;
  flowId?: string;
}

const Widget: React.FC<WidgetProps> = ({
  step,
  selected = false,
  onClick,
  mode = "list",
  style,
  className = "",
  execution,
  satisfiedArgs = new Set(),
  attributeProvenance = new Map(),
  attributeValues,
  flowId,
}) => {
  const stepType = getStepType(step);

  const widgetClassName = [
    "step-widget",
    stepType,
    mode,
    selected && "selected",
    onClick && "clickable",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={widgetClassName} style={style} onClick={onClick}>
      <StepHeader step={step} />
      <Attributes
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
        attributeProvenance={attributeProvenance}
        attributeValues={attributeValues}
      />
      <StepPredicate step={step} />
      <Footer step={step} flowId={flowId} execution={execution} />
    </div>
  );
};

export default React.memo(Widget);
