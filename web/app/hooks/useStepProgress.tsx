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
  flowId?: string;
  workItems?: WorkItemProgress;
}

export const useStepProgress = (
  stepId: string,
  flowId?: string,
  execution?: ExecutionResult
) => {
  const { events } = useWebSocketContext();
  const executions = useExecutions();
  const [progressState, setProgressState] = useState<StepProgressState>({
    status: "pending",
  });

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

  useEffect(() => {
    if (!flowId) {
      setProgressState({ status: "pending" });
      return;
    }

    if (
      execution &&
      execution.flow_id === flowId &&
      execution.step_id === stepId
    ) {
      setProgressState({
        status: execution.status as StepProgressStatus,
        flowId,
        workItems: workItemProgress,
      });
      return;
    }

    const stepEvents = events.filter(
      (event) =>
        event.data?.step_id === stepId && event.data?.flow_id === flowId
    );

    if (stepEvents.length === 0) {
      setProgressState({
        status: "pending",
        flowId,
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
          flowId,
          workItems: workItemProgress,
        };
        break;
      case "step_completed":
        newState = {
          status: "completed",
          startTime: latestEvent.data?.start_time,
          endTime: latestEvent.data?.end_time,
          flowId,
          workItems: workItemProgress,
        };
        break;
      case "step_failed":
        newState = {
          status: "failed",
          startTime: latestEvent.data?.start_time,
          endTime: latestEvent.data?.end_time,
          flowId,
          workItems: workItemProgress,
        };
        break;
      default:
        newState = {
          status: "pending",
          flowId,
          workItems: workItemProgress,
        };
    }

    setProgressState(newState);
  }, [events, stepId, flowId, execution, workItemProgress]);

  return progressState;
};
