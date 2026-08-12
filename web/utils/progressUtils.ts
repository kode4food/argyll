import {
  IconCompensate,
  IconCompensateFailed,
  IconProgressActive,
  IconProgressCompleted,
  IconProgressFailed,
  IconProgressPending,
  IconProgressSkipped,
  type LucideIcon,
} from "./iconRegistry";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";

const progressIconMap: Record<StepProgressStatus, LucideIcon> = {
  pending: IconProgressPending,
  active: IconProgressActive,
  compensating: IconProgressActive,
  completed: IconProgressCompleted,
  failed: IconProgressFailed,
  skipped: IconProgressSkipped,
  compensated: IconCompensate,
  compensation_failed: IconCompensateFailed,
};

export const getProgressIcon = (status: StepProgressStatus): LucideIcon => {
  return progressIconMap[status] ?? IconProgressPending;
};
