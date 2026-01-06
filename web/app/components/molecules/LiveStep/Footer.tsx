import React from "react";
import { Step, ExecutionResult } from "@/app/api";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "@/app/hooks/useStepProgress";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import Tooltip from "@/app/components/atoms/Tooltip";
import styles from "../StepShared/StepFooter.module.css";
import {
  formatScriptPreview,
  getScriptIcon,
  getHttpIcon,
  getSkipReason,
} from "@/utils/stepFooterUtils";
import { useMemo } from "react";

interface FooterProps {
  step: Step;
  flowId?: string;
  execution?: ExecutionResult;
}

const Footer: React.FC<FooterProps> = ({ step, flowId, execution }) => {
  const progressState = useStepProgress(step.id, flowId, execution);
  const { displayInfo, tooltipSections } = useMemo(() => {
    let displayInfo: {
      icon: React.ComponentType<any>;
      text: string;
      className?: string;
    } | null = null;

    if (step.type === "script" && step.script) {
      const ScriptIcon = getScriptIcon(step.script.language);
      const scriptPreview = formatScriptPreview(step.script.script);
      displayInfo = {
        icon: ScriptIcon,
        text: scriptPreview,
        className: "endpoint-script",
      };
    } else if (step.http) {
      const HttpIcon = getHttpIcon(step.type);
      displayInfo = {
        icon: HttpIcon,
        text: step.http.endpoint,
      };
    }

    const sections: React.ReactElement[] = [];
    if (!execution || !flowId) {
      return { displayInfo, tooltipSections: sections };
    }

    const StatusIcon = getProgressIcon(execution.status);
    const workItemSuffix = progressState.workItems
      ? (() => {
          const done = progressState.workItems.completed;
          const failed = progressState.workItems.failed;
          const total = progressState.workItems.total;
          const base = `${done} of ${total}`;
          return failed > 0 ? ` (${base}, ${failed} failed)` : ` (${base})`;
        })()
      : "";

    sections.push(
      <TooltipSection
        key="execution-status"
        title="Execution Status"
        icon={
          <StatusIcon
            className={`progress-icon ${execution.status || "pending"}`}
          />
        }
        bold
      >
        {execution.status.toUpperCase()}
        {workItemSuffix}
      </TooltipSection>
    );

    if (execution.status === "failed" && execution.error_message) {
      sections.push(
        <TooltipSection key="error" title="Error" monospace>
          {execution.error_message}
        </TooltipSection>
      );
    }

    if (execution.status === "skipped") {
      const skipReason = getSkipReason(step);
      sections.push(
        <TooltipSection key="reason" title="Reason">
          {skipReason}
        </TooltipSection>
      );
    }

    if (execution.status === "completed" && execution.duration_ms) {
      sections.push(
        <TooltipSection key="duration" title="Duration">
          {execution.duration_ms}ms
        </TooltipSection>
      );
    }

    return { displayInfo, tooltipSections: sections };
  }, [execution, flowId, progressState, step]);

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
