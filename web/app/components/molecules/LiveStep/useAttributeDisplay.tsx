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

/**
 * Hook that returns a function to render status badges for attributes
 * Encapsulates conditional logic for different attribute types and statuses
 */
export const useAttributeStatusBadge = () => {
  return useMemo(() => {
    return (
      argType: ArgType,
      context: StatusBadgeContext
    ): React.ReactElement | null => {
      const {
        isSatisfied,
        executionStatus,
        isWinner,
        isProvidedByUpstream,
        wasDefaulted,
      } = context;

      if (argType === "optional" && executionStatus) {
        if (executionStatus === "skipped") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
            >
              <IconAttributeStatusSkipped className={styles.statusIcon} />
            </div>
          );
        }
        if (isProvidedByUpstream) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <IconAttributeStatusProvided className={styles.statusIcon} />
            </div>
          );
        }
        if (wasDefaulted) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.defaulted} arg-status-badge defaulted`}
            >
              <IconAttributeStatusDefaulted className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
          >
            <IconAttributeStatusPending className={styles.statusIcon} />
          </div>
        );
      }

      if (argType === "const" && executionStatus) {
        if (executionStatus === "skipped") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
            >
              <IconAttributeStatusSkipped className={styles.statusIcon} />
            </div>
          );
        }
        if (wasDefaulted || isSatisfied) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.defaulted} arg-status-badge defaulted`}
            >
              <IconAttributeStatusDefaulted className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
          >
            <IconAttributeStatusPending className={styles.statusIcon} />
          </div>
        );
      }

      if (argType === "required") {
        if (isSatisfied) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <IconAttributeStatusSatisfied className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "failed" || executionStatus === "skipped") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.failed} arg-status-badge failed`}
            >
              <IconAttributeStatusFailed className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
          >
            <IconAttributeStatusPending className={styles.statusIcon} />
          </div>
        );
      }

      if (argType === "output") {
        if (executionStatus === "skipped" || executionStatus === "failed") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
            >
              <IconAttributeStatusBlocked className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "active") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
            >
              <IconAttributeStatusPending className={styles.statusIcon} />
            </div>
          );
        }
        if (isWinner) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <IconAttributeStatusWinner className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "completed") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.notWinner} arg-status-badge not-winner`}
            >
              <IconAttributeStatusNotWinner className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.placeholder} arg-status-badge placeholder`}
          />
        );
      }

      return null;
    };
  }, []);
};
