import { FlowContext, FlowStatus } from "@/app/api";
import { StepProgressStatus } from "@/app/hooks/useStepProgress";
import { sanitizeFlowID } from "@/utils/flowUtils";

export type SelectableFlow = Pick<FlowContext, "id" | "status">;

export function mapFlowStatusToProgressStatus(
  status: FlowStatus
): StepProgressStatus {
  switch (status) {
    case "pending":
      return "pending";
    case "active":
      return "active";
    case "completed":
      return "completed";
    case "failed":
      return "failed";
    default:
      return "pending";
  }
}

export function filterFlowsBySearch<T extends SelectableFlow>(
  flows: T[],
  searchTerm: string
): T[] {
  const sanitized = sanitizeFlowID(searchTerm);
  return flows.filter((flow) => flow.id.includes(sanitized));
}
