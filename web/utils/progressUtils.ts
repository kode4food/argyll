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
      return Clock;
    case "active":
      return Loader2;
    case "completed":
      return CheckCircle;
    case "failed":
      return XCircle;
    case "skipped":
      return MinusCircle;
    default:
      return Clock;
  }
};
