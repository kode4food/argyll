import React from "react";
import { Step } from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "./StepHeader.module.css";
import { useT } from "@/app/i18n";
import { IconMemoizable } from "@/utils/iconRegistry";

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
      {step.memoizable && (
        <span
          className={`step-type-icon ${styles.memoIcon}`}
          aria-label={t("stepEditor.memoizableLabel")}
          title={t("stepEditor.memoizableTitle")}
        >
          <IconMemoizable aria-hidden="true" />
        </span>
      )}
    </div>
  );
};

export default React.memo(StepHeader);
