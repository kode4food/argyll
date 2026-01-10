import React from "react";
import { Step } from "@/app/api";
import StepTypeLabel from "@/app/components/atoms/StepTypeLabel";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "./StepHeader.module.css";
import { useT } from "@/app/i18n";

interface StepHeaderProps {
  step: Step;
}

const StepHeader: React.FC<StepHeaderProps> = ({ step }) => {
  const t = useT();

  return (
    <div className={`${styles.header} step-header`}>
      <Tooltip
        trigger={
          <div className={styles.titleContainer}>
            <h3 className={styles.title}>{step.name}</h3>
          </div>
        }
      >
        <TooltipSection title={t("tooltip.stepName")}>
          {step.name}
        </TooltipSection>
        <TooltipSection title={t("tooltip.stepId")}>{step.id}</TooltipSection>
      </Tooltip>
      <StepTypeLabel step={step} />
    </div>
  );
};

export default React.memo(StepHeader);
