import React, { useEffect, useMemo, useState } from "react";
import { Step, ExecutionResult, WorkState } from "@/app/api";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "@/app/hooks/useStepProgress";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import Tooltip from "@/app/components/atoms/Tooltip";
import styles from "../StepShared/StepFooter.module.css";
import { formatScriptPreview } from "@/utils/stepFooterUtils";
import { getStepTypeIcon, IconMemoizable } from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";

interface FooterProps {
  step: Step;
  flowId?: string;
  execution?: ExecutionResult;
}

const MS_PER_SECOND = 1000;
const SECONDS_PER_MINUTE = 60;
const COMPLETE_PERCENT = 100;

const formatRemaining = (ms: number): string => {
  const totalSeconds = Math.max(0, Math.ceil(ms / MS_PER_SECOND));
  const minutes = Math.floor(totalSeconds / SECONDS_PER_MINUTE);
  const seconds = totalSeconds % SECONDS_PER_MINUTE;
  if (minutes === 0) return `${seconds}s`;
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
};

const parseTime = (value?: string): number | undefined => {
  if (!value) return undefined;
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? undefined : parsed;
};

interface WorkTimerProps {
  start: number;
  end: number;
}

const WorkTimer: React.FC<WorkTimerProps> = ({ start, end }) => {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = window.setInterval(
      () => setNow(Date.now()),
      MS_PER_SECOND
    );
    return () => window.clearInterval(interval);
  }, []);

  const duration = Math.max(1, end - start);
  const elapsed = Math.max(0, Math.min(duration, now - start));
  const remaining = Math.max(0, end - now);
  const value = Math.round((elapsed / duration) * COMPLETE_PERCENT);

  return (
    <div className={styles.workTimer}>
      <progress className={styles.workTimerProgress} max={100} value={value} />
      <span className={styles.workTimerRemaining}>
        {formatRemaining(remaining)}
      </span>
    </div>
  );
};

const workTimerTiming = (work: WorkState, step: Step) => {
  if (work.status === "active" && step.http?.timeout && work.started_at) {
    const start = parseTime(work.started_at);
    if (!start) return undefined;
    return {
      titleKey: "liveStep.activeTimeoutTitle",
      start,
      end: start + step.http.timeout,
    };
  }

  if (work.status === "pending" && work.next_retry_at) {
    const end = parseTime(work.next_retry_at);
    const start = parseTime(work.completed_at) ?? parseTime(work.started_at);
    if (!start || !end) return undefined;
    return { titleKey: "liveStep.retryCountdownTitle", start, end };
  }

  return undefined;
};

const Footer: React.FC<FooterProps> = ({ step, flowId, execution }) => {
  const t = useT();
  const progressState = useStepProgress(step.id, flowId, execution);
  const { displayInfo, tooltipSections } = useMemo(() => {
    let displayInfo: {
      icon: React.ComponentType<any>;
      text: string;
      className?: string;
    } | null = null;

    const TypeIcon = getStepTypeIcon(step.type);

    if (step.type === "script" && step.script) {
      const scriptPreview = formatScriptPreview(step.script.script);
      displayInfo = {
        icon: TypeIcon,
        text: scriptPreview,
        className: "endpoint-script",
      };
    } else if (step.type === "flow" && step.flow?.goals?.length) {
      displayInfo = {
        icon: TypeIcon,
        text: step.flow.goals.join(", "),
      };
    } else if (step.http) {
      const method = step.http.method || "POST";
      displayInfo = {
        icon: TypeIcon,
        text: `${method} ${step.http.endpoint}`,
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

    Object.entries(execution.work_items ?? {}).forEach(([token, work]) => {
      const timing = workTimerTiming(work, step);
      if (!timing) return;
      sections.push(
        <TooltipSection key={`work-timer-${token}`} title={t(timing.titleKey)}>
          <WorkTimer start={timing.start} end={timing.end} />
        </TooltipSection>
      );
    });

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
            {step.memoizable && (
              <div className={styles.memoIcon}>
                <IconMemoizable className={styles.icon} />
              </div>
            )}
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
