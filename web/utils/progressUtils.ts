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

const PROGRESS_ICON_MAP: Record<StepProgressStatus, LucideIcon> = {
  pending: IconProgressPending,
  active: IconProgressActive,
  compensating: IconProgressActive,
  completed: IconProgressCompleted,
  failed: IconProgressFailed,
  skipped: IconProgressSkipped,
  compensated: IconCompensate,
  compensate_failed: IconCompensateFailed,
};

export const getProgressIcon = (status: StepProgressStatus): LucideIcon => {
  return PROGRESS_ICON_MAP[status] ?? IconProgressPending;
};
