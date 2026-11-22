import React from "react";
import { Step } from "@/app/api";
import { getArgIcon } from "@/utils/iconRegistry";
import styles from "./FlowStats.module.css";
import { useT } from "@/app/i18n";
import { useFlowStats } from "./useFlowStats";

interface FlowStatsProps {
  steps: Step[];
  executionSequence: string[];
  resolvedAttributes: string[];
}

const FlowStats: React.FC<FlowStatsProps> = React.memo(function FlowStats({
  steps,
  executionSequence,
  resolvedAttributes,
}) {
  const t = useT();
  const stats = useFlowStats(steps, executionSequence, resolvedAttributes);

  const RequiredIcon = getArgIcon("required").Icon;
  const OptionalIcon = getArgIcon("optional").Icon;
  const OutputIcon = getArgIcon("output").Icon;

  return (
    <div className={`${styles.root} flow-stats`}>
      {stats.requiredInputs > 0 && (
        <span
          className={`status-bubble ${styles.badge} ${styles.badgeRequired} stat-badge stat-badge--required`}
        >
          <RequiredIcon className={styles.badgeIcon} />
          {t("common.countOf", {
            resolved: stats.resolvedRequired,
            total: stats.requiredInputs,
          })}
        </span>
      )}
      {stats.optionalInputs > 0 && (
        <span
          className={`status-bubble ${styles.badge} ${styles.badgeOptional} stat-badge stat-badge--optional`}
        >
          <OptionalIcon className={styles.badgeIcon} />
          {t("common.countOf", {
            resolved: stats.resolvedOptional,
            total: stats.optionalInputs,
          })}
        </span>
      )}
      {stats.outputs > 0 && (
        <span
          className={`status-bubble ${styles.badge} ${styles.badgeOutput} stat-badge stat-badge--output`}
        >
          <OutputIcon className={styles.badgeIcon} />
          {t("common.countOf", {
            resolved: stats.resolvedOutputs,
            total: stats.outputs,
          })}
        </span>
      )}
    </div>
  );
});

export default FlowStats;
