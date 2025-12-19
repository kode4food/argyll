import React, { useMemo } from "react";
import {
  CheckCircle2,
  XCircle,
  Award,
  Ban,
  CircleDashed,
  CheckCircle,
  CircleDot,
  CircleSlash,
} from "lucide-react";
import { ArgType, StatusBadgeContext } from "./stepAttributesSectionUtils";
import styles from "../StepAttributesSection.module.css";

/**
 * Hook that returns a function to render status badges for attributes
 * Encapsulates complex conditional logic for different attribute types and statuses
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
              <CircleSlash className={styles.statusIcon} />
            </div>
          );
        }
        if (isProvidedByUpstream) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <CheckCircle className={styles.statusIcon} />
            </div>
          );
        }
        if (wasDefaulted) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.defaulted} arg-status-badge defaulted`}
            >
              <CircleDot className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
          >
            <CircleDashed className={styles.statusIcon} />
          </div>
        );
      }

      if (argType === "required") {
        if (isSatisfied) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <CheckCircle2 className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "failed" || executionStatus === "skipped") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.failed} arg-status-badge failed`}
            >
              <XCircle className={styles.statusIcon} />
            </div>
          );
        }
        return (
          <div
            className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
          >
            <CircleDashed className={styles.statusIcon} />
          </div>
        );
      }

      if (argType === "output") {
        if (executionStatus === "skipped" || executionStatus === "failed") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.skipped} arg-status-badge skipped`}
            >
              <Ban className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "active") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.pending} arg-status-badge pending`}
            >
              <CircleDashed className={styles.statusIcon} />
            </div>
          );
        }
        if (isWinner) {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.satisfied} arg-status-badge satisfied`}
            >
              <Award className={styles.statusIcon} />
            </div>
          );
        }
        if (executionStatus === "completed") {
          return (
            <div
              className={`${styles.argStatusBadge} ${styles.notWinner} arg-status-badge not-winner`}
            >
              <XCircle className={styles.statusIcon} />
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
