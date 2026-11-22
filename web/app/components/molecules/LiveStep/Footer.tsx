import React from "react";
import { Step, ExecutionResult } from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import styles from "@/app/components/molecules/StepShared/StepFooter.module.css";
import StepInfoDisplay from "@/app/components/molecules/StepShared/StepInfoDisplay";
import { useFooter } from "./useFooter";

interface FooterProps {
  step: Step;
  flowId?: string;
  execution?: ExecutionResult;
}

const Footer: React.FC<FooterProps> = ({ step, flowId, execution }) => {
  const {
    displayInfo,
    tooltipSections,
    hasFlowProgress,
    ProgressIcon,
    progressStatus,
  } = useFooter(step, flowId, execution);

  return (
    <Tooltip
      trigger={
        <div className={`${styles.footer} step-footer`}>
          <StepInfoDisplay displayInfo={displayInfo} />
          <div className={styles.actions}>
            <div className={styles.healthStatus}>
              {hasFlowProgress ? (
                <div className={styles.progressContainer}>
                  <ProgressIcon
                    className={`progress-icon ${progressStatus || "pending"}`}
                  />
                </div>
              ) : null}
            </div>
          </div>
        </div>
      }
    >
      <>{tooltipSections}</>
    </Tooltip>
  );
};

export default Footer;
