import React from "react";
import { Step } from "@/app/api";
import { getStepType } from "@/utils/stepUtils";
import { getStepActionIcon } from "@/utils/iconRegistry";

interface StepTypeLabelProps {
  step: Step;
}

const StepTypeLabel: React.FC<StepTypeLabelProps> = ({ step }) => {
  const stepType = getStepType(step);
  const TypeIcon = getStepActionIcon(step);

  return (
    <span className={`step-type-label ${stepType}`} aria-label={step.type}>
      <TypeIcon aria-hidden="true" />
    </span>
  );
};

export default StepTypeLabel;
