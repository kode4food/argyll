import React from "react";
import { Step, ExecutionResult } from "@/app/api";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "@/app/hooks/useStepProgress";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "@/app/components/molecules/StepShared/StepFooter.module.css";
import { flowGoalsText } from "@/app/components/molecules/StepShared/stepDisplayInfo";
import WorkTimer, { workTimerTiming } from "./WorkTimer";

const USAGE_SIGNIFICANT_DIGITS = 10;

const skippedReason = (
  execution: ExecutionResult,
  step: Step,
  t: (key: string, vars?: Record<string, string | number>) => string
): string => {
  switch (execution.error_message) {
    case "predicate returned false":
      return t("liveStep.skipPredicate");
    case "required match did not match":
      return t("liveStep.skipRequiredMatch");
    case "outputs not needed":
      return t("liveStep.skipOutputsNotNeeded");
    case undefined:
    case "":
      return t("liveStep.skipUnknown");
    default:
      return execution.error_message;
  }
};

interface ExecutionContext {
  execution: ExecutionResult | undefined;
  flowId: string | undefined;
  locale: string;
  progressState: ReturnType<typeof useStepProgress>;
}

const agentUsageTable = (
  items: ExecutionResult["work_items"],
  locale: string,
  t: (key: string) => string
) => {
  const total = new Map<string, number>();
  for (const work of Object.values(items ?? {})) {
    for (const [name, value] of Object.entries(work.usage ?? {})) {
      total.set(name, (total.get(name) ?? 0) + value);
    }
  }
  if (total.size === 0) return null;
  const rows = [...total].sort(
    ([left], [right]) => Number(right === "cost") - Number(left === "cost")
  );
  return (
    <TooltipSection key="usage" title={t("liveStep.usageTitle")}>
      <table
        className={styles.usageTable}
        aria-label={t("liveStep.usageTitle")}
      >
        <tbody>
          {rows.map(([name, value]) => {
            const key = `liveStep.usage.${name}`;
            const label = t(key);
            return (
              <tr key={name}>
                <th scope="row">{label === key ? name : label}</th>
                <td>
                  {value.toLocaleString(
                    locale,
                    Number.isInteger(value)
                      ? undefined
                      : { maximumSignificantDigits: USAGE_SIGNIFICANT_DIGITS }
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </TooltipSection>
  );
};

export const buildTooltipSections = (
  step: Step,
  ctx: ExecutionContext,
  t: (key: string, vars?: Record<string, string | number>) => string
): React.ReactElement[] => {
  const { execution, flowId, locale, progressState } = ctx;
  const sections: React.ReactElement[] = [];
  const goals = flowGoalsText(step);

  if (goals) {
    sections.push(
      <TooltipSection key="goals" title={t("stepFooter.flowGoals")}>
        {goals}
      </TooltipSection>
    );
  }

  if (!execution || !flowId) {
    return sections;
  }

  const StatusIcon = getProgressIcon(progressState.status);
  const workItemSuffix = progressState.workItems
    ? (() => {
        const done = progressState.workItems.completed;
        const failed = progressState.workItems.failed;
        const total = progressState.workItems.total;
        const base =
          failed > 0
            ? t("liveStep.workItemsSummaryFailed", { done, total, failed })
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
          className={`progress-icon ${progressState.status || "pending"}`}
        />
      }
      bold
    >
      {progressState.status.toUpperCase()}
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
    sections.push(
      <TooltipSection key="reason" title={t("liveStep.reasonTitle")}>
        {skippedReason(execution, step, t)}
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

  if (step.type === "agent") {
    const usage = agentUsageTable(execution.work_items, locale, t);
    if (usage) sections.push(usage);
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

  return sections;
};
