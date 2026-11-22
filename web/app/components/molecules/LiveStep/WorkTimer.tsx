import React from "react";
import { Step, WorkState } from "@/app/api";
import styles from "@/app/components/molecules/StepShared/StepFooter.module.css";
import { COMPLETE_PERCENT, MS_PER_SECOND, useWorkTimer } from "./useWorkTimer";

const SECONDS_PER_MINUTE = 60;

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
  const { value, remaining } = useWorkTimer(start, end);

  return (
    <div className={styles.workTimer}>
      <progress
        className={styles.workTimerProgress}
        max={COMPLETE_PERCENT}
        value={value}
      />
      <span className={styles.workTimerRemaining}>
        {formatRemaining(remaining)}
      </span>
    </div>
  );
};

export const workTimerTiming = (work: WorkState, step: Step) => {
  if (
    work.status === "active" &&
    step.http?.invoke?.timeout &&
    work.started_at
  ) {
    const start = parseTime(work.started_at);
    if (!start) return undefined;
    return {
      titleKey: "liveStep.activeTimeoutTitle",
      start,
      end: start + step.http.invoke.timeout,
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

export default WorkTimer;
