import React from "react";
import type { Step } from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "./StepHeader.module.css";
import { useT } from "@/app/i18n";
import { IconMemoized } from "@/utils/iconRegistry";

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
      {step.handling === "memoized" && (
        <span
          className={`step-type-icon ${styles.memoIcon}`}
          aria-label={t("stepEditor.handling.memoized")}
          title={t("stepEditor.handling.memoized")}
        >
          <IconMemoized aria-hidden="true" />
        </span>
      )}
    </div>
  );
};

export default React.memo(StepHeader);
