import React from "react";
import { Step } from "../../api";
import { getStepType, getStepTypeLabel } from "@/utils/stepUtils";

interface StepTypeLabelProps {
  step: Step;
  className?: string;
}

const StepTypeLabel: React.FC<StepTypeLabelProps> = ({
  step,
  className = "",
}) => {
  const stepType = getStepType(step);
  const label = getStepTypeLabel(stepType);

  return (
    <span className={`step-type-label ${stepType} ${className}`}>{label}</span>
  );
};

export default StepTypeLabel;
