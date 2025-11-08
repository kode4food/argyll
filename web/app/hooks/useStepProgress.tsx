import { useState, useEffect, useMemo } from "react";
import { useWebSocketContext } from "./useWebSocketContext";
import { ExecutionResult, StepStatus } from "../api";
import { useExecutions } from "../store/workflowStore";

export type StepProgressStatus = StepStatus;

interface WorkItemProgress {
  total: number;
  completed: number;
  failed: number;
  active: number;
}

interface StepProgressState {
  status: StepProgressStatus;
  startTime?: number;
  endTime?: number;
  workflowId?: string;
  workItems?: WorkItemProgress;
}

export const useStepProgress = (
  stepId: string,
  workflowId?: string,
  execution?: ExecutionResult
) => {
  const { events } = useWebSocketContext();
  const executions = useExecutions();
  const [progressState, setProgressState] = useState<StepProgressState>({
    status: "pending",
  });

  // Calculate work item progress from executions
  const workItemProgress = useMemo(() => {
    if (!workflowId || !executions) return undefined;

    // Find the execution for this step
    const exec = executions.find(
      (e: any) => e.step_id === stepId && e.workflow_id === workflowId
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
  }, [executions, stepId, workflowId]);

  useEffect(() => {
    if (!workflowId) {
      setProgressState({ status: "pending" });
      return;
    }

    if (
      execution &&
      execution.workflow_id === workflowId &&
      execution.step_id === stepId
    ) {
      setProgressState({
        status: execution.status as StepProgressStatus,
        workflowId,
        workItems: workItemProgress,
      });
      return;
    }

    const stepEvents = events.filter(
      (event) =>
        event.data?.step_id === stepId && event.data?.workflow_id === workflowId
    );

    if (stepEvents.length === 0) {
      setProgressState({
        status: "pending",
        workflowId,
        workItems: workItemProgress,
      });
      return;
    }

    const latestEvent = stepEvents[stepEvents.length - 1];

    let newState: StepProgressState;

    switch (latestEvent.type) {
      case "step_started":
        newState = {
          status: "active",
          startTime: latestEvent.data?.start_time,
          workflowId,
          workItems: workItemProgress,
        };
        break;
      case "step_completed":
        newState = {
          status: "completed",
          startTime: latestEvent.data?.start_time,
          endTime: latestEvent.data?.end_time,
          workflowId,
          workItems: workItemProgress,
        };
        break;
      case "step_failed":
        newState = {
          status: "failed",
          startTime: latestEvent.data?.start_time,
          endTime: latestEvent.data?.end_time,
          workflowId,
          workItems: workItemProgress,
        };
        break;
      default:
        newState = {
          status: "pending",
          workflowId,
          workItems: workItemProgress,
        };
    }

    setProgressState(newState);
  }, [events, stepId, workflowId, execution, workItemProgress]);

  return progressState;
};
