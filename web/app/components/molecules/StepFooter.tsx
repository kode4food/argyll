import React from "react";
import { Step, ExecutionResult, HealthStatus } from "../../api";
import { getHealthIconClass, getHealthStatusText } from "@/utils/healthUtils";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "../../hooks/useStepProgress";
import { useStepFooterDisplay } from "./StepFooter/useStepFooterDisplay";
import Tooltip from "../atoms/Tooltip";
import HealthDot from "../atoms/HealthDot";
import styles from "./StepFooter.module.css";

interface StepFooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
  flowId?: string;
  execution?: ExecutionResult;
}

const StepFooter: React.FC<StepFooterProps> = ({
  step,
  healthStatus,
  healthError,
  flowId,
  execution,
}) => {
  const progressState = useStepProgress(step.id, flowId, execution);
  const healthIconClass = getHealthIconClass(healthStatus, step.type);
  const healthText = getHealthStatusText(healthStatus, healthError);
  const { displayInfo, tooltipSections } = useStepFooterDisplay(
    step,
    execution,
    flowId,
    healthStatus,
    healthError,
    healthText,
    progressState
  );

  const useProgress = flowId && progressState.flowId === flowId;
  const ProgressIcon = getProgressIcon(progressState.status);

  return (
    <Tooltip
      trigger={
        <div className={`${styles.footer} step-footer`}>
          {displayInfo && (
            <div className={styles.infoDisplay}>
              {React.createElement(displayInfo.icon, {
                className: `step-type-icon ${styles.icon}`,
              })}
              <span
                className={`${styles.endpoint} ${
                  displayInfo.className === "endpoint-script"
                    ? styles.endpointScript
                    : ""
                } step-endpoint`}
              >
                {displayInfo.text}
              </span>
            </div>
          )}
          <div className={styles.actions}>
            <div className={styles.healthStatus}>
              {useProgress ? (
                <div className={styles.progressContainer}>
                  <ProgressIcon
                    className={`progress-icon ${progressState.status || "pending"}`}
                  />
                  {progressState.status === "active" &&
                    progressState.workItems &&
                    progressState.workItems.total > 1 && (
                      <span className={styles.progressCounter}>
                        (
                        {progressState.workItems.completed +
                          progressState.workItems.failed}
                        /{progressState.workItems.total})
                      </span>
                    )}
                </div>
              ) : (
                <HealthDot status={healthIconClass as HealthStatus} />
              )}
            </div>
          </div>
        </div>
      }
    >
      <>{tooltipSections}</>
    </Tooltip>
  );
};

export default StepFooter;
