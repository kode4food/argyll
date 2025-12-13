import React from "react";
import { Step } from "../../api";
import StepTypeLabel from "../atoms/StepTypeLabel";
import Tooltip from "../atoms/Tooltip";
import TooltipSection from "../atoms/TooltipSection";
import styles from "./StepHeader.module.css";

interface StepHeaderProps {
  step: Step;
}

const StepHeader: React.FC<StepHeaderProps> = ({ step }) => {
  return (
    <div className={`${styles.header} step-header`}>
      <Tooltip
        trigger={
          <div className={styles.titleContainer}>
            <h3 className={styles.title}>{step.name}</h3>
          </div>
        }
      >
        <TooltipSection title="Step Name">{step.name}</TooltipSection>
        <TooltipSection title="Step ID">{step.id}</TooltipSection>
      </Tooltip>
      <StepTypeLabel step={step} />
    </div>
  );
};

export default React.memo(StepHeader);
