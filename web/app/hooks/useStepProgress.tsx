import { useMemo } from "react";
import { ExecutionResult, StepStatus } from "../api";
import { useExecutions } from "../store/flowStore";

export type StepProgressStatus = StepStatus;

interface WorkItemProgress {
  total: number;
  completed: number;
  failed: number;
  active: number;
}

interface StepProgressState {
  status: StepProgressStatus;
  flowId?: string;
  workItems?: WorkItemProgress;
}

export const useStepProgress = (
  stepId: string,
  flowId?: string,
  execution?: ExecutionResult
) => {
  const executions = useExecutions();

  // Calculate work item progress from executions
  const workItemProgress = useMemo(() => {
    if (!flowId || !executions) return undefined;

    // Find the execution for this step
    const exec = executions.find(
      (e: any) => e.step_id === stepId && e.flow_id === flowId
    );

    if (!exec || !exec.work_items) return undefined;

    const items = Object.values(exec.work_items);
    if (items.length === 0) return undefined;

    const progress: WorkItemProgress = {
      total: items.length,
      completed: items.filter((item: any) => item.status === "completed")
        .length,
      failed: items.filter((item: any) => item.status === "failed").length,
      active: items.filter((item: any) => item.status === "active").length,
    };

    return progress;
  }, [executions, stepId, flowId]);

  const effectiveExecution = useMemo(() => {
    if (
      execution &&
      execution.flow_id === flowId &&
      execution.step_id === stepId
    ) {
      return execution;
    }
    if (!flowId) return undefined;
    return executions.find(
      (e: any) => e.step_id === stepId && e.flow_id === flowId
    );
  }, [execution, executions, flowId, stepId]);

  const status = effectiveExecution?.status
    ? (effectiveExecution.status as StepProgressStatus)
    : "pending";

  const progressState: StepProgressState = {
    status,
    flowId,
    workItems: workItemProgress,
  };

  return progressState;
};
