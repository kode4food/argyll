import React, { useMemo } from "react";
import {
  IconAttributeStatusBlocked,
  IconAttributeStatusDefaulted,
  IconAttributeStatusFailed,
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

const skippedBadge = (): React.ReactElement =>
  badge("skipped", IconAttributeStatusSkipped);

const optionalStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (!context.executionStatus) return null;
  if (context.executionStatus === "skipped") return skippedBadge();
  if (context.isProvidedByUpstream)
    return badge("satisfied", IconAttributeStatusProvided);
  if (context.wasDefaulted)
    return badge("defaulted", IconAttributeStatusDefaulted);
  return pendingBadge();
};

const constStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (!context.executionStatus) return null;
  if (context.executionStatus === "skipped") return skippedBadge();
  if (context.wasDefaulted || context.isSatisfied)
    return badge("defaulted", IconAttributeStatusDefaulted);
  return pendingBadge();
};

const requiredStatusBadge = (
  context: StatusBadgeContext
): React.ReactElement | null => {
  if (context.isSatisfied)
    return badge("satisfied", IconAttributeStatusSatisfied);
  if (
    context.executionStatus === "failed" ||
    context.executionStatus === "skipped"
  )
    return badge("failed", IconAttributeStatusFailed);
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
            return constStatusBadge(context);
          case "required":
            return requiredStatusBadge(context);
          case "output":
            return outputStatusBadge(context);
          default:
            return null;
        }
      },
    []
  );
};
