"use client";

import React, { useCallback, useEffect, useRef } from "react";
import { useWebSocketClient } from "@/app/hooks/useWebSocketClient";
import { useFlowStore } from "@/app/store/flowStore";
import { FlowContext } from "@/app/api";
import { WebSocketEvent } from "@/app/types/websocket";

const ENGINE_EVENT_TYPES = [
  "step_registered",
  "step_unregistered",
  "step_updated",
  "step_health_changed",
  "flow_activated",
  "flow_deactivated",
];

const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const selectedFlow = useFlowStore((state) => state.selectedFlow);
  const loadFlows = useFlowStore((state) => state.loadFlows);
  const addStep = useFlowStore((state) => state.addStep);
  const updateStep = useFlowStore((state) => state.updateStep);
  const removeStep = useFlowStore((state) => state.removeStep);
  const updateStepHealth = useFlowStore((state) => state.updateStepHealth);
  const setEngineSocketStatus = useFlowStore(
    (state) => state.setEngineSocketStatus
  );
  const engineReconnectRequest = useFlowStore(
    (state) => state.engineReconnectRequest
  );

  const handleEngineEvent = useCallback(
    (event: WebSocketEvent) => {
      switch (event.type) {
        case "step_registered": {
          const step = event.data?.step;
          if (step) addStep(step);
          break;
        }
        case "step_unregistered": {
          const stepId = event.data?.step_id;
          if (stepId) removeStep(stepId);
          break;
        }
        case "step_updated": {
          const step = event.data?.step;
          if (step) updateStep(step);
          break;
        }
        case "step_health_changed": {
          const stepId = event.data?.step_id;
          const health = event.data?.status;
          const error = event.data?.error;
          if (stepId && health) {
            updateStepHealth(stepId, health, error);
          }
          break;
        }
        case "flow_activated":
        case "flow_deactivated":
          loadFlows();
          break;
        default:
          break;
      }
    },
    [addStep, removeStep, updateStep, updateStepHealth, loadFlows]
  );

  const handleFlowEvent = useCallback((event: WebSocketEvent) => {
    const {
      selectedFlow: activeFlowId,
      flowData,
      initializeExecutions,
      updateExecution,
      updateFlowFromWebSocket,
    } = useFlowStore.getState();

    if (!activeFlowId || event.data?.flow_id !== activeFlowId || !flowData) {
      return;
    }

    const flowUpdate: Partial<FlowContext> = {};

    switch (event.type) {
      case "flow_started":
        flowUpdate.status = "active";
        flowUpdate.started_at =
          event.data?.started_at || new Date().toISOString();
        if (event.data?.plan) {
          initializeExecutions(activeFlowId, event.data.plan);
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
          flowUpdate.state = {
            ...(flowData.state || {}),
            [key]: { value, step: stepId },
          };
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
        break;
    }

    if (Object.keys(flowUpdate).length > 0) {
      updateFlowFromWebSocket(flowUpdate);
    }
  }, []);

  const engineClient = useWebSocketClient({
    enabled: true,
    onEvent: handleEngineEvent,
  });
  const flowClient = useWebSocketClient({
    enabled: Boolean(selectedFlow),
    onEvent: handleFlowEvent,
  });

  useEffect(() => {
    engineClient.subscribe({
      aggregate_id: ["engine"],
      event_types: ENGINE_EVENT_TYPES,
    });
  }, [engineClient.subscribe]);

  useEffect(() => {
    if (selectedFlow) {
      flowClient.subscribe({
        aggregate_id: ["flow", selectedFlow],
      });
    }
  }, [selectedFlow, flowClient.subscribe]);

  useEffect(() => {
    setEngineSocketStatus(
      engineClient.connectionStatus,
      engineClient.reconnectAttempt
    );
  }, [
    engineClient.connectionStatus,
    engineClient.reconnectAttempt,
    setEngineSocketStatus,
  ]);

  const engineReconnectRef = useRef(engineReconnectRequest);
  useEffect(() => {
    if (engineReconnectRequest === engineReconnectRef.current) {
      return;
    }
    engineReconnectRef.current = engineReconnectRequest;
    engineClient.reconnect();
  }, [engineReconnectRequest, engineClient.reconnect]);

  return <>{children}</>;
};

export default WebSocketProvider;
