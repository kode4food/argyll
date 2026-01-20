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
  getFlowIcon,
} from "@/utils/stepFooterUtils";
import { useMemo } from "react";
import { useT } from "@/app/i18n";

interface FooterProps {
  step: Step;
  flowId?: string;
  execution?: ExecutionResult;
}

const Footer: React.FC<FooterProps> = ({ step, flowId, execution }) => {
  const t = useT();
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
    } else if (step.type === "flow" && step.flow?.goals?.length) {
      const FlowIcon = getFlowIcon();
      displayInfo = {
        icon: FlowIcon,
        text: step.flow.goals.join(", "),
      };
    } else if (step.http) {
      const HttpIcon = getHttpIcon(step.type);
      displayInfo = {
        icon: HttpIcon,
        text: step.http.endpoint,
      };
    }

    const sections: React.ReactElement[] = [];
    if (step.type === "flow" && step.flow?.goals?.length) {
      sections.push(
        <TooltipSection key="goals" title={t("stepFooter.flowGoals")}>
          {step.flow.goals.join(", ")}
        </TooltipSection>
      );
    }
    if (!execution || !flowId) {
      return { displayInfo, tooltipSections: sections };
    }

    const StatusIcon = getProgressIcon(execution.status);
    const workItemSuffix = progressState.workItems
      ? (() => {
          const done = progressState.workItems.completed;
          const failed = progressState.workItems.failed;
          const total = progressState.workItems.total;
          const base =
            failed > 0
              ? t("liveStep.workItemsSummaryFailed", {
                  done,
                  total,
                  failed,
                })
              : t("liveStep.workItemsSummary", { done, total });
          return ` (${base})`;
        })()
      : "";

    sections.push(
      <TooltipSection
        key="execution-status"
        title={t("liveStep.executionStatus")}
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
        <TooltipSection key="error" title={t("liveStep.errorTitle")} monospace>
          {execution.error_message}
        </TooltipSection>
      );
    }

    if (execution.status === "skipped") {
      const skipReason = step.predicate
        ? t("liveStep.skipPredicate")
        : t("liveStep.skipMissingInputs");
      sections.push(
        <TooltipSection key="reason" title={t("liveStep.reasonTitle")}>
          {skipReason}
        </TooltipSection>
      );
    }

    if (execution.status === "completed" && execution.duration_ms) {
      sections.push(
        <TooltipSection key="duration" title={t("liveStep.durationTitle")}>
          {t("common.durationMs", { duration: execution.duration_ms })}
        </TooltipSection>
      );
    }

    return { displayInfo, tooltipSections: sections };
  }, [execution, flowId, progressState, step, t]);

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
