import { useEffect, useRef } from "react";
import { useWebSocketContext } from "@/app/hooks/useWebSocketContext";
import { useFlowStore } from "@/app/store/flowStore";
import { FlowContext } from "@/app/api";

type FlowEvent = {
  type: string;
  id?: string[];
  sequence?: number;
  timestamp?: string | number;
  data?: Record<string, any>;
};

const isNewEvent = (event: FlowEvent, seenSequences: Map<string, number>) => {
  const key = event.id?.join(":") || "";
  const currentMax = seenSequences.get(key) || 0;
  const sequence = event.sequence || 0;
  if (sequence <= currentMax) {
    return false;
  }
  seenSequences.set(key, sequence);
  return true;
};

const handleStepCatalogEvent = (
  event: FlowEvent,
  handlers: {
    addStep: (step: any) => void;
    removeStep: (id: string) => void;
    updateStep: (step: any) => void;
    updateStepHealth: (id: string, health: string, error?: string) => void;
  }
) => {
  switch (event.type) {
    case "step_registered": {
      const step = event.data?.step;
      if (step) handlers.addStep(step);
      return true;
    }
    case "step_unregistered": {
      const stepId = event.data?.step_id;
      if (stepId) handlers.removeStep(stepId);
      return true;
    }
    case "step_updated": {
      const step = event.data?.step;
      if (step) handlers.updateStep(step);
      return true;
    }
    case "step_health_changed": {
      const stepId = event.data?.step_id;
      const health = event.data?.status;
      const error = event.data?.error;
      if (stepId && health) {
        handlers.updateStepHealth(stepId, health, error);
      }
      return true;
    }
    default:
      return false;
  }
};

const handleFlowEvent = (
  event: FlowEvent,
  context: {
    selectedFlow: string | null;
    flowData: FlowContext | null;
    initializeExecutions: (flowId: string, plan: any) => void;
    updateExecution: (stepId: string, update: any) => void;
  }
) => {
  const { selectedFlow, flowData, initializeExecutions, updateExecution } =
    context;
  if (!selectedFlow || event.data?.flow_id !== selectedFlow || !flowData) {
    return null;
  }

  const flowUpdate: Partial<FlowContext> = {};
  const stateUpdates: Record<string, { value: any; step: string }> = {};

  switch (event.type) {
    case "flow_started":
      flowUpdate.status = "active";
      flowUpdate.started_at =
        event.data?.started_at || new Date().toISOString();
      if (event.data?.plan) {
        initializeExecutions(selectedFlow, event.data.plan);
      }
      break;
    case "step_started":
      updateExecution(event.data?.step_id, {
        status: "active",
        inputs: event.data?.inputs,
        work_items: event.data?.work_items || {},
        started_at: new Date(event.timestamp || Date.now()).toISOString(),
      });
      break;
    case "step_completed":
      updateExecution(event.data?.step_id, {
        status: "completed",
        outputs: event.data?.outputs,
        duration_ms: event.data?.duration,
        completed_at: new Date(event.timestamp || Date.now()).toISOString(),
      });
      break;
    case "step_failed":
      updateExecution(event.data?.step_id, {
        status: "failed",
        error_message: event.data?.error,
        completed_at: new Date(event.timestamp || Date.now()).toISOString(),
      });
      break;
    case "step_skipped":
      updateExecution(event.data?.step_id, {
        status: "skipped",
        completed_at: new Date(event.timestamp || Date.now()).toISOString(),
      });
      break;
    case "attribute_set": {
      const key = event.data?.key;
      const value = event.data?.value;
      const stepId = event.data?.step_id;
      if (key && value !== undefined) {
        stateUpdates[key] = { value, step: stepId };
      }
      break;
    }
    case "flow_completed":
      flowUpdate.status = "completed";
      flowUpdate.completed_at =
        event.data?.completed_at || new Date().toISOString();
      break;
    case "flow_failed":
      flowUpdate.status = "failed";
      flowUpdate.error_state = {
        message: event.data?.error || "Flow failed",
        step_id: "",
        timestamp: event.data?.failed_at || new Date().toISOString(),
      };
      flowUpdate.completed_at =
        event.data?.failed_at || new Date().toISOString();
      break;
    default:
      return null;
  }

  return { flowUpdate, stateUpdates };
};

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
  const updateStep = useFlowStore((state) => state.updateStep);
  const removeStep = useFlowStore((state) => state.removeStep);
  const initializeExecutions = useFlowStore(
    (state) => state.initializeExecutions
  );
  const updateExecution = useFlowStore((state) => state.updateExecution);

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
      if (!event) continue;
      if (!isNewEvent(event, seenSequences.current)) continue;

      const handled = handleStepCatalogEvent(event, {
        addStep,
        removeStep,
        updateStep,
        updateStepHealth,
      });
      if (handled) continue;

      const flowResult = handleFlowEvent(event, {
        selectedFlow,
        flowData,
        initializeExecutions,
        updateExecution,
      });

      if (flowResult?.flowUpdate) {
        flowUpdate = { ...flowUpdate, ...flowResult.flowUpdate };
      }
      if (flowResult?.stateUpdates) {
        stateUpdates = { ...stateUpdates, ...flowResult.stateUpdates };
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
    updateStep,
    removeStep,
    initializeExecutions,
    updateExecution,
    updateConsumerCursor,
  ]);
};
