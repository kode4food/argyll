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

const computeWorkItemProgress = (
  executions: ExecutionResult[],
  stepId: string,
  flowId: string
): WorkItemProgress | undefined => {
  const exec = executions.find(
    (e: any) => e.step_id === stepId && e.flow_id === flowId
  );
  if (!exec || !exec.work_items) return undefined;
  const items = Object.values(exec.work_items);
  if (items.length === 0) return undefined;
  return {
    total: items.length,
    completed: items.filter((item: any) => item.status === "succeeded").length,
    failed: items.filter((item: any) => item.status === "failed").length,
    active: items.filter((item: any) => item.status === "active").length,
  };
};

interface FlowStep {
  stepId: string;
  flowId: string | undefined;
}

const findEffectiveExecution = (
  executions: ExecutionResult[],
  key: FlowStep,
  execution: ExecutionResult | undefined
): ExecutionResult | undefined => {
  const { stepId, flowId } = key;
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
};

export const useStepProgress = (
  stepId: string,
  flowId?: string,
  execution?: ExecutionResult
) => {
  const executions = useExecutions();

  const workItemProgress = useMemo(
    () =>
      flowId ? computeWorkItemProgress(executions, stepId, flowId) : undefined,
    [executions, stepId, flowId]
  );

  const effectiveExecution = useMemo(
    () => findEffectiveExecution(executions, { stepId, flowId }, execution),
    [execution, executions, flowId, stepId]
  );

  const status = effectiveExecution?.status
    ? (effectiveExecution.status as StepProgressStatus)
    : "pending";

  return useMemo<StepProgressState>(
    () => ({
      status,
      flowId,
      workItems: workItemProgress,
    }),
    [status, flowId, workItemProgress]
  );
};
