"use client";

import React, { useCallback, useEffect, useRef } from "react";
import { useWebSocketClient } from "@/app/hooks/useWebSocketClient";
import { useFlowStore } from "@/app/store/flowStore";
import { FlowContext } from "@/app/api";
import { WebSocketEvent, WebSocketSubscribeState } from "@/app/types/websocket";

const ENGINE_EVENT_TYPES = [
  "step_registered",
  "step_unregistered",
  "step_updated",
  "step_health_changed",
  "flow_activated",
  "flow_deactivated",
];

const FLOW_EVENT_TYPES = [
  "flow_started",
  "step_started",
  "step_completed",
  "step_failed",
  "step_skipped",
  "attribute_set",
  "flow_completed",
  "flow_failed",
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
    (event: WebSocketEvent | WebSocketSubscribeState) => {
      if (event.type === "subscribe_state") {
        const { setEngineState } = useFlowStore.getState();
        setEngineState((event as WebSocketSubscribeState).data as any);
        return;
      }

      const wsEvent = event as WebSocketEvent;
      switch (wsEvent.type) {
        case "step_registered": {
          const step = wsEvent.data?.step;
          if (step) addStep(step);
          break;
        }
        case "step_unregistered": {
          const stepId = wsEvent.data?.step_id;
          if (stepId) removeStep(stepId);
          break;
        }
        case "step_updated": {
          const step = wsEvent.data?.step;
          if (step) updateStep(step);
          break;
        }
        case "step_health_changed": {
          const stepId = wsEvent.data?.step_id;
          const health = wsEvent.data?.status;
          const error = wsEvent.data?.error;
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

  const handleFlowEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribeState) => {
      if (event.type === "subscribe_state") {
        const { setFlowState } = useFlowStore.getState();
        setFlowState((event as WebSocketSubscribeState).data as any);
        return;
      }

      const wsEvent = event as WebSocketEvent;
      const {
        selectedFlow: activeFlowId,
        flowData,
        initializeExecutions,
        updateExecution,
        updateFlowFromWebSocket,
      } = useFlowStore.getState();

      if (
        !activeFlowId ||
        wsEvent.data?.flow_id !== activeFlowId ||
        !flowData
      ) {
        return;
      }

      const flowUpdate: Partial<FlowContext> = {};

      switch (wsEvent.type) {
        case "flow_started":
          flowUpdate.status = "active";
          flowUpdate.started_at =
            wsEvent.data?.started_at || new Date().toISOString();
          if (wsEvent.data?.plan) {
            initializeExecutions(activeFlowId, wsEvent.data.plan);
          }
          break;
        case "step_started":
          updateExecution(wsEvent.data?.step_id, {
            status: "active",
            inputs: wsEvent.data?.inputs,
            work_items: wsEvent.data?.work_items || {},
            started_at: new Date(wsEvent.timestamp || Date.now()).toISOString(),
          });
          break;
        case "step_completed":
          updateExecution(wsEvent.data?.step_id, {
            status: "completed",
            outputs: wsEvent.data?.outputs,
            duration_ms: wsEvent.data?.duration,
            completed_at: new Date(
              wsEvent.timestamp || Date.now()
            ).toISOString(),
          });
          break;
        case "step_failed":
          updateExecution(wsEvent.data?.step_id, {
            status: "failed",
            error_message: wsEvent.data?.error,
            completed_at: new Date(
              wsEvent.timestamp || Date.now()
            ).toISOString(),
          });
          break;
        case "step_skipped":
          updateExecution(wsEvent.data?.step_id, {
            status: "skipped",
            completed_at: new Date(
              wsEvent.timestamp || Date.now()
            ).toISOString(),
          });
          break;
        case "attribute_set": {
          const key = wsEvent.data?.key;
          const value = wsEvent.data?.value;
          const stepId = wsEvent.data?.step_id;
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
            wsEvent.data?.completed_at || new Date().toISOString();
          break;
        case "flow_failed":
          flowUpdate.status = "failed";
          flowUpdate.error_state = {
            message: wsEvent.data?.error || "Flow failed",
            step_id: "",
            timestamp: wsEvent.data?.failed_at || new Date().toISOString(),
          };
          flowUpdate.completed_at =
            wsEvent.data?.failed_at || new Date().toISOString();
          break;
        default:
          break;
      }

      if (Object.keys(flowUpdate).length > 0) {
        updateFlowFromWebSocket(flowUpdate);
      }
    },
    []
  );

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
        event_types: FLOW_EVENT_TYPES,
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
