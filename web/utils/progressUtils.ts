import {
  Clock,
  Loader2,
  CheckCircle,
  XCircle,
  MinusCircle,
} from "lucide-react";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";

export const getProgressIcon = (status: StepProgressStatus) => {
  switch (status) {
    case "pending":
      return Clock; // Clock for waiting
    case "active":
      return Loader2; // Better spinning loader
    case "completed":
      return CheckCircle; // Success checkmark
    case "failed":
      return XCircle; // Error X
    case "skipped":
      return MinusCircle; // Skipped
    default:
      return Clock;
  }
};

export const getProgressIconClass = (
  status: StepProgressStatus | null | undefined
) => {
  return status || "pending";
};
