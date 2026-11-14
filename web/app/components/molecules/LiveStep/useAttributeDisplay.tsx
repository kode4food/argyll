import React, { useMemo } from "react";
import {
  IconAttributeStatusBlocked,
  IconAttributeStatusDefaulted,
  IconAttributeStatusFailed,
  IconAttributeStatusFlowStart,
  IconAttributeStatusNotWinner,
  IconAttributeStatusPending,
  IconAttributeStatusProvided,
  IconAttributeStatusSatisfied,
  IconAttributeStatusSkipped,
  IconAttributeStatusWinner,
} from "@/utils/iconRegistry";
import { ArgType, StatusBadgeContext } from "./attributeUtils";
import styles from "../StepShared/StepAttributesSection.module.css";

const badge = (
  statusClass: string,
  Icon: React.ComponentType<{ className?: string }>
): React.ReactElement => (
  <div
    className={`${styles.argStatusBadge} ${styles[statusClass]} arg-status-badge ${statusClass}`}
  >
    <Icon className={styles.statusIcon} />
  </div>
);

const pendingBadge = (): React.ReactElement =>
  badge("pending", IconAttributeStatusPending);

const progressBadge = (): React.ReactElement =>
  badge("progress", IconAttributeStatusPending);

const skippedBadge = (): React.ReactElement =>
  badge("skipped", IconAttributeStatusSkipped);

const failedBadge = (): React.ReactElement =>
  badge("failed", IconAttributeStatusFailed);

const optionalStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (!context.executionStatus || context.executionStatus === "pending") {
    return context.isAvailable ? progressBadge() : pendingBadge();
  }
  if (context.isUnsatisfied) return failedBadge();
  if (context.executionStatus === "skipped" && !context.isSatisfied)
    return skippedBadge();
  if (context.isProvidedByUpstream)
    return badge("satisfied", IconAttributeStatusProvided);
  if (context.wasDefaulted)
    return badge("defaulted", IconAttributeStatusDefaulted);
  if (context.isSatisfied)
    return badge("satisfied", IconAttributeStatusSatisfied);
  if (context.isAvailable) return progressBadge();
  return pendingBadge();
};

// The engine supplies these values, so nothing upstream is ever awaited
const flowStartStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null =>
  context.executionStatus === "skipped"
    ? skippedBadge()
    : badge("flowStart", IconAttributeStatusFlowStart);

const requiredStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (!context.executionStatus || context.executionStatus === "pending") {
    return context.isAvailable ? progressBadge() : pendingBadge();
  }
  if (context.isUnsatisfied) return failedBadge();
  if (context.isSatisfied)
    return badge("satisfied", IconAttributeStatusSatisfied);
  if (
    context.executionStatus === "failed" ||
    context.executionStatus === "skipped"
  )
    return failedBadge();
  if (context.isAvailable) return progressBadge();
  return pendingBadge();
};

const outputStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (
    context.executionStatus === "skipped" ||
    context.executionStatus === "failed"
  )
    return badge("skipped", IconAttributeStatusBlocked);
  if (context.executionStatus === "active") return pendingBadge();
  if (context.isWinner) return badge("satisfied", IconAttributeStatusWinner);
  if (context.executionStatus === "completed")
    return (
      <div
        className={`${styles.argStatusBadge} ${styles.notWinner} arg-status-badge not-winner`}
      >
        <IconAttributeStatusNotWinner className={styles.statusIcon} />
      </div>
    );
  return (
    <div
      className={`${styles.argStatusBadge} ${styles.placeholder} arg-status-badge placeholder`}
    />
  );
};

export const useAttributeStatusBadge = () => {
  return useMemo(
    () =>
      (
        argType: ArgType,
        context: StatusBadgeContext
      ): React.ReactElement | null => {
        switch (argType) {
          case "optional":
            return optionalStatusBadge(context);
          case "const":
          case "meta":
          case "feed":
            return flowStartStatusBadge(context);
          case "required":
            return requiredStatusBadge(context);
          case "output":
          case "metric":
            return outputStatusBadge(context);
          default:
            return null;
        }
      },
    []
  );
};
