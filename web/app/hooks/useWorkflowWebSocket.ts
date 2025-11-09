import { useEffect, useRef } from "react";
import { useWebSocketContext } from "./useWebSocketContext";
import { useWorkflowStore } from "../store/workflowStore";
import { WorkflowContext } from "../api";

export const useWorkflowWebSocket = () => {
  const {
    events,
    subscribe,
    registerConsumer,
    unregisterConsumer,
    updateConsumerCursor,
  } = useWebSocketContext();
  const selectedWorkflow = useWorkflowStore((state) => state.selectedWorkflow);
  const nextSequence = useWorkflowStore((state) => state.nextSequence);
  const workflowData = useWorkflowStore((state) => state.workflowData);
  const refreshExecutions = useWorkflowStore(
    (state) => state.refreshExecutions
  );
  const updateWorkflow = useWorkflowStore(
    (state) => state.updateWorkflowFromWebSocket
  );
  const updateStepHealth = useWorkflowStore((state) => state.updateStepHealth);
  const addStep = useWorkflowStore((state) => state.addStep);
  const removeStep = useWorkflowStore((state) => state.removeStep);

  const consumerIdRef = useRef<string | null>(null);
  useEffect(() => {
    consumerIdRef.current = registerConsumer();
    return () => {
      if (consumerIdRef.current) {
        unregisterConsumer(consumerIdRef.current);
      }
    };
  }, [registerConsumer, unregisterConsumer]);

  useEffect(() => {
    if (selectedWorkflow) {
      subscribe({
        engine_events: true,
        flow_id: selectedWorkflow,
        from_sequence: nextSequence,
      });
    } else {
      subscribe({ engine_events: true });
    }
  }, [selectedWorkflow, nextSequence, subscribe]);

  const lastProcessedEventIndex = useRef(-1);
  const seenSequences = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    if (!events.length) return;

    const newEvents = events.slice(lastProcessedEventIndex.current + 1);
    if (!newEvents.length) return;

    let needsExecutionRefresh = false;
    let stateUpdates: Record<string, { value: any; step: string }> = {};
    let workflowUpdate: Partial<WorkflowContext> = {};

    for (const event of newEvents) {
      if (!event) {
        continue;
      }

      const aggregateKey = event.aggregate_id?.join(":") || "";
      const currentMax = seenSequences.current.get(aggregateKey) || 0;
      if (event.sequence <= currentMax) {
        continue;
      }
      seenSequences.current.set(aggregateKey, event.sequence);

      if (event.type === "step_registered") {
        const step = event.data?.step;
        if (step) {
          addStep(step);
        }
        continue;
      }

      if (event.type === "step_unregistered") {
        const stepId = event.data?.step_id;
        if (stepId) {
          removeStep(stepId);
        }
        continue;
      }

      if (event.type === "step_health_changed") {
        const stepId = event.data?.step_id;
        const health = event.data?.health_status;
        const error = event.data?.health_error;
        if (stepId && health) {
          updateStepHealth(stepId, health, error);
        }
        continue;
      }

      if (!selectedWorkflow) continue;
      if (event.data?.flow_id !== selectedWorkflow) continue;
      if (!workflowData) continue;

      if (event.type === "workflow_started") {
        workflowUpdate.status = "active";
        workflowUpdate.started_at =
          event.data?.started_at || new Date().toISOString();
      } else if (event.type === "step_started") {
        needsExecutionRefresh = true;
      } else if (event.type === "step_completed") {
        needsExecutionRefresh = true;
      } else if (event.type === "step_failed") {
        needsExecutionRefresh = true;
      } else if (event.type === "step_skipped") {
        needsExecutionRefresh = true;
      } else if (event.type === "attribute_set") {
        const key = event.data?.key;
        const value = event.data?.value;
        const stepId = event.data?.step_id;
        if (key && value !== undefined) {
          stateUpdates[key] = { value, step: stepId };
        }
        needsExecutionRefresh = true;
      } else if (event.type === "workflow_completed") {
        workflowUpdate.status = "completed";
        workflowUpdate.completed_at =
          event.data?.completed_at || new Date().toISOString();
      } else if (event.type === "workflow_failed") {
        workflowUpdate.status = "failed";
        workflowUpdate.error_state = {
          message: event.data?.error || "Workflow failed",
          step_id: "",
          timestamp: event.data?.failed_at || new Date().toISOString(),
        };
        workflowUpdate.completed_at =
          event.data?.failed_at || new Date().toISOString();
      }
    }

    if (Object.keys(stateUpdates).length > 0 && workflowData) {
      workflowUpdate.state = {
        ...workflowData.state,
        ...stateUpdates,
      };
    }

    if (Object.keys(workflowUpdate).length > 0) {
      updateWorkflow(workflowUpdate);
    }

    lastProcessedEventIndex.current = events.length - 1;

    if (consumerIdRef.current && events.length > 0) {
      updateConsumerCursor(
        consumerIdRef.current,
        lastProcessedEventIndex.current + 1
      );
    }

    if (needsExecutionRefresh && selectedWorkflow) {
      refreshExecutions(selectedWorkflow);
    }
  }, [
    events,
    selectedWorkflow,
    workflowData,
    updateWorkflow,
    updateStepHealth,
    addStep,
    removeStep,
    refreshExecutions,
    updateConsumerCursor,
  ]);
};
