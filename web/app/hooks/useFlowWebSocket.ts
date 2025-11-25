import { useEffect, useRef } from "react";
import { useWebSocketContext } from "./useWebSocketContext";
import { useFlowStore } from "../store/flowStore";
import { FlowContext } from "../api";

export const useFlowWebSocket = () => {
  const {
    events,
    subscribe,
    registerConsumer,
    unregisterConsumer,
    updateConsumerCursor,
  } = useWebSocketContext();
  const selectedFlow = useFlowStore((state) => state.selectedFlow);
  const nextSequence = useFlowStore((state) => state.nextSequence);
  const flowData = useFlowStore((state) => state.flowData);
  const updateFlow = useFlowStore((state) => state.updateFlowFromWebSocket);
  const updateStepHealth = useFlowStore((state) => state.updateStepHealth);
  const addStep = useFlowStore((state) => state.addStep);
  const removeStep = useFlowStore((state) => state.removeStep);
  const addOrUpdateExecution = useFlowStore(
    (state) => state.addOrUpdateExecution
  );

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
    if (selectedFlow) {
      subscribe({
        engine_events: true,
        flow_id: selectedFlow,
        from_sequence: nextSequence,
      });
    } else {
      subscribe({ engine_events: true });
    }
  }, [selectedFlow, nextSequence, subscribe]);

  const lastProcessedEventIndex = useRef(-1);
  const seenSequences = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    if (!events.length) return;

    const newEvents = events.slice(lastProcessedEventIndex.current + 1);
    if (!newEvents.length) return;

    let stateUpdates: Record<string, { value: any; step: string }> = {};
    let flowUpdate: Partial<FlowContext> = {};

    for (const event of newEvents) {
      if (!event) {
        continue;
      }

      const key = event.id?.join(":") || "";
      const currentMax = seenSequences.current.get(key) || 0;
      if (event.sequence <= currentMax) {
        continue;
      }
      seenSequences.current.set(key, event.sequence);

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
        const health = event.data?.status;
        const error = event.data?.error;
        if (stepId && health) {
          updateStepHealth(stepId, health, error);
        }
        continue;
      }

      if (!selectedFlow) continue;
      if (event.data?.flow_id !== selectedFlow) continue;
      if (!flowData) continue;

      if (event.type === "flow_started") {
        flowUpdate.status = "active";
        flowUpdate.started_at =
          event.data?.started_at || new Date().toISOString();
      } else if (event.type === "step_started") {
        addOrUpdateExecution({
          step_id: event.data?.step_id,
          flow_id: selectedFlow,
          status: "active",
          inputs: event.data?.inputs,
          work_items: event.data?.work_items || {},
          started_at: new Date(event.timestamp).toISOString(),
        });
      } else if (event.type === "step_completed") {
        addOrUpdateExecution({
          step_id: event.data?.step_id,
          flow_id: selectedFlow,
          status: "completed",
          outputs: event.data?.outputs,
          duration_ms: event.data?.duration,
          completed_at: new Date(event.timestamp).toISOString(),
        });
      } else if (event.type === "step_failed") {
        addOrUpdateExecution({
          step_id: event.data?.step_id,
          flow_id: selectedFlow,
          status: "failed",
          error_message: event.data?.error,
          completed_at: new Date(event.timestamp).toISOString(),
        });
      } else if (event.type === "step_skipped") {
        addOrUpdateExecution({
          step_id: event.data?.step_id,
          flow_id: selectedFlow,
          status: "skipped",
          completed_at: new Date(event.timestamp).toISOString(),
        });
      } else if (event.type === "attribute_set") {
        const key = event.data?.key;
        const value = event.data?.value;
        const stepId = event.data?.step_id;
        if (key && value !== undefined) {
          stateUpdates[key] = { value, step: stepId };
        }
      } else if (event.type === "flow_completed") {
        flowUpdate.status = "completed";
        flowUpdate.completed_at =
          event.data?.completed_at || new Date().toISOString();
      } else if (event.type === "flow_failed") {
        flowUpdate.status = "failed";
        flowUpdate.error_state = {
          message: event.data?.error || "Flow failed",
          step_id: "",
          timestamp: event.data?.failed_at || new Date().toISOString(),
        };
        flowUpdate.completed_at =
          event.data?.failed_at || new Date().toISOString();
      }
    }

    if (Object.keys(stateUpdates).length > 0 && flowData) {
      flowUpdate.state = {
        ...flowData.state,
        ...stateUpdates,
      };
    }

    if (Object.keys(flowUpdate).length > 0) {
      updateFlow(flowUpdate);
    }

    lastProcessedEventIndex.current = events.length - 1;

    if (consumerIdRef.current && events.length > 0) {
      updateConsumerCursor(
        consumerIdRef.current,
        lastProcessedEventIndex.current + 1
      );
    }
  }, [
    events,
    selectedFlow,
    flowData,
    updateFlow,
    updateStepHealth,
    addStep,
    removeStep,
    addOrUpdateExecution,
    updateConsumerCursor,
  ]);
};
