import React from "react";
import { Step } from "@/app/api";
import { getStepType, getStepTypeLabel } from "@/utils/stepUtils";

interface StepTypeLabelProps {
  step: Step;
}

const StepTypeLabel: React.FC<StepTypeLabelProps> = ({ step }) => {
  const stepType = getStepType(step);
  const label = getStepTypeLabel(stepType);

  return <span className={`step-type-label ${stepType}`}>{label}</span>;
};

export default StepTypeLabel;
