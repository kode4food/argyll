import React, { useCallback, useEffect, useRef } from "react";
import { useWebSocketClient } from "@/app/hooks/useWebSocketClient";
import { useFlowStore } from "@/app/store/flowStore";
import { FlowContext } from "@/app/api";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import { useT } from "@/app/i18n";

const CATALOG_EVENT_TYPES = [
  "step_registered",
  "step_unregistered",
  "step_updated",
];

const PARTITION_EVENT_TYPES = [
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
  "work_started",
  "work_succeeded",
  "work_failed",
  "work_not_completed",
  "retry_scheduled",
];

const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const t = useT();
  const selectedFlow = useFlowStore((state) => state.selectedFlow);
  const loadSteps = useFlowStore((state) => state.loadSteps);
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

  const handleCatalogEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setCatalogState } = useFlowStore.getState();
        const data = (event as WebSocketSubscribed).data as any;
        setCatalogState(data?.steps ?? {});
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
        default:
          break;
      }
    },
    [addStep, removeStep, updateStep]
  );

  const handlePartitionEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setPartitionState } = useFlowStore.getState();
        const data = (event as WebSocketSubscribed).data as any;
        setPartitionState(data?.health ?? {});
        return;
      }

      const wsEvent = event as WebSocketEvent;
      switch (wsEvent.type) {
        case "step_health_changed": {
          const stepId = wsEvent.data?.step_id;
          const health = wsEvent.data?.status;
          const error = wsEvent.data?.error;
          if (stepId && health) {
            updateStepHealth(stepId, health, error);
            loadSteps();
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
    [updateStepHealth, loadSteps, loadFlows]
  );

  const handleFlowEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setFlowState } = useFlowStore.getState();
        setFlowState((event as WebSocketSubscribed).data as any);
        return;
      }

      const wsEvent = event as WebSocketEvent;
      const {
        selectedFlow: activeFlowId,
        flowData,
        initializeExecutions,
        updateExecution,
        updateWorkItem,
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
            message: wsEvent.data?.error || t("flow.failed"),
            step_id: "",
            timestamp: wsEvent.data?.failed_at || new Date().toISOString(),
          };
          flowUpdate.completed_at =
            wsEvent.data?.failed_at || new Date().toISOString();
          break;
        case "work_started":
          updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
            status: "active",
            inputs: wsEvent.data?.inputs,
          });
          break;
        case "work_succeeded":
          updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
            status: "succeeded",
            outputs: wsEvent.data?.outputs,
          });
          break;
        case "work_failed":
          updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
            status: "failed",
            error: wsEvent.data?.error,
          });
          break;
        case "work_not_completed":
          updateWorkItem(
            wsEvent.data?.step_id,
            wsEvent.data?.token,
            {
              status: "not_completed",
              error: wsEvent.data?.error,
            },
            wsEvent.data?.retry_token
          );
          break;
        case "retry_scheduled":
          updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
            status: "pending",
            retry_count: wsEvent.data?.retry_count ?? 0,
            next_retry_at: wsEvent.data?.next_retry_at,
            error: wsEvent.data?.error,
          });
          break;
        default:
          break;
      }

      if (Object.keys(flowUpdate).length > 0) {
        updateFlowFromWebSocket(flowUpdate);
      }
    },
    [t]
  );

  const catalogClient = useWebSocketClient({
    enabled: true,
    onEvent: handleCatalogEvent,
  });
  const partitionClient = useWebSocketClient({
    enabled: true,
    onEvent: handlePartitionEvent,
  });
  const flowClient = useWebSocketClient({
    enabled: Boolean(selectedFlow),
    onEvent: handleFlowEvent,
  });

  useEffect(() => {
    catalogClient.subscribe({
      aggregate_id: ["catalog"],
      event_types: CATALOG_EVENT_TYPES,
    });
  }, [catalogClient.subscribe]);

  useEffect(() => {
    partitionClient.subscribe({
      aggregate_id: ["partition"],
      event_types: PARTITION_EVENT_TYPES,
    });
  }, [partitionClient.subscribe]);

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
      catalogClient.connectionStatus,
      catalogClient.reconnectAttempt
    );
  }, [
    catalogClient.connectionStatus,
    catalogClient.reconnectAttempt,
    setEngineSocketStatus,
  ]);

  const engineReconnectRef = useRef(engineReconnectRequest);
  useEffect(() => {
    if (engineReconnectRequest === engineReconnectRef.current) {
      return;
    }
    engineReconnectRef.current = engineReconnectRequest;
    catalogClient.reconnect();
    partitionClient.reconnect();
  }, [
    engineReconnectRequest,
    catalogClient.reconnect,
    partitionClient.reconnect,
  ]);

  return <>{children}</>;
};

export default WebSocketProvider;
