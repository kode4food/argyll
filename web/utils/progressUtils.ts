import {
  IconProgressActive,
  IconProgressCompleted,
  IconProgressFailed,
  IconProgressPending,
  IconProgressSkipped,
} from "./iconRegistry";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";

export const getProgressIcon = (status: StepProgressStatus) => {
  switch (status) {
    case "pending":
      return IconProgressPending;
    case "active":
      return IconProgressActive;
    case "completed":
      return IconProgressCompleted;
    case "failed":
      return IconProgressFailed;
    case "skipped":
      return IconProgressSkipped;
    default:
      return IconProgressPending;
  }
};
