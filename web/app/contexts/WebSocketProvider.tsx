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

const CLUSTER_EVENT_TYPES = ["step_health_changed"];

const FLOW_SUMMARY_EVENT_TYPES = [
  "flow_started",
  "flow_completed",
  "flow_failed",
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

const eventTimestamp = (timestamp?: number): string => {
  return new Date(timestamp || Date.now()).toISOString();
};

const subscribedData = (event: WebSocketSubscribed): any => {
  return event.items[0]?.data;
};

const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const t = useT();
  const selectedFlow = useFlowStore((state) => state.selectedFlow);
  const visibleFlowIDs = useFlowStore((state) => state.visibleFlowIDs);
  const addFlow = useFlowStore((state) => state.addFlow);
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
        const data = subscribedData(event as WebSocketSubscribed);
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

  const handleClusterEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setHealthState } = useFlowStore.getState();
        const data = subscribedData(event as WebSocketSubscribed);
        const nodes: Record<string, any> = data?.nodes ?? {};
        const healthByNode: Record<string, Record<string, any>> = {};
        for (const [nodeId, node] of Object.entries(nodes)) {
          if (node && typeof node === "object" && "health" in node) {
            healthByNode[nodeId] = (node as any).health ?? {};
          }
        }
        setHealthState(healthByNode);
        return;
      }

      const wsEvent = event as WebSocketEvent;
      switch (wsEvent.type) {
        case "step_health_changed": {
          const nodeId = wsEvent.data?.node_id;
          const stepId = wsEvent.data?.step_id;
          const health = wsEvent.data?.status;
          const error = wsEvent.data?.error;
          if (nodeId && stepId && health) {
            updateStepHealth(nodeId, stepId, health, error);
          }
          break;
        }
        default:
          break;
      }
    },
    [updateStepHealth]
  );

  const handleFlowSummaryEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        return;
      }

      const wsEvent = event as WebSocketEvent;
      const flowId = wsEvent.data?.flow_id;
      if (!flowId) {
        return;
      }

      const timestamp = eventTimestamp(wsEvent.timestamp);

      switch (wsEvent.type) {
        case "flow_started":
          addFlow({
            id: flowId,
            status: "active",
            timestamp,
          });
          break;
        case "flow_completed":
          addFlow({
            id: flowId,
            status: "completed",
            timestamp,
          });
          break;
        case "flow_failed":
          addFlow({
            id: flowId,
            status: "failed",
            timestamp,
            error: wsEvent.data?.error || t("flow.failed"),
          });
          break;
        default:
          break;
      }
    },
    [addFlow, t]
  );

  const handleFlowEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setFlowNotFound, setFlowState } = useFlowStore.getState();
        const data = subscribedData(event as WebSocketSubscribed);
        if (!data) {
          if (selectedFlow) {
            setFlowNotFound(selectedFlow);
          }
          return;
        }
        setFlowState(data);
        return;
      }

      const wsEvent = event as WebSocketEvent;
      const {
        selectedFlow: activeFlowId,
        flowData,
        initializeExecutions,
        updateExecution,
        updateWorkItem,
        updateFlowData,
        addFlow,
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
          addFlow({
            id: activeFlowId,
            status: "active",
            timestamp: eventTimestamp(wsEvent.timestamp),
          });
          flowUpdate.status = "active";
          flowUpdate.started_at = eventTimestamp(wsEvent.timestamp);
          if (wsEvent.data?.plan) {
            initializeExecutions(activeFlowId, wsEvent.data.plan);
          }
          break;
        case "step_started":
          updateExecution(wsEvent.data?.step_id, {
            status: "active",
            inputs: wsEvent.data?.inputs,
            work_items: wsEvent.data?.work_items || {},
            started_at: eventTimestamp(wsEvent.timestamp),
          });
          break;
        case "step_completed":
          updateExecution(wsEvent.data?.step_id, {
            status: "completed",
            outputs: wsEvent.data?.outputs,
            duration_ms: wsEvent.data?.duration,
            completed_at: eventTimestamp(wsEvent.timestamp),
          });
          break;
        case "step_failed":
          updateExecution(wsEvent.data?.step_id, {
            status: "failed",
            error_message: wsEvent.data?.error,
            completed_at: eventTimestamp(wsEvent.timestamp),
          });
          break;
        case "step_skipped":
          updateExecution(wsEvent.data?.step_id, {
            status: "skipped",
            completed_at: eventTimestamp(wsEvent.timestamp),
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
          addFlow({
            id: activeFlowId,
            status: "completed",
            timestamp: eventTimestamp(wsEvent.timestamp),
          });
          flowUpdate.status = "completed";
          flowUpdate.completed_at = eventTimestamp(wsEvent.timestamp);
          break;
        case "flow_failed":
          const failedAt = eventTimestamp(wsEvent.timestamp);
          addFlow({
            id: activeFlowId,
            status: "failed",
            timestamp: failedAt,
            error: wsEvent.data?.error || t("flow.failed"),
          });
          flowUpdate.status = "failed";
          flowUpdate.error_state = {
            message: wsEvent.data?.error || t("flow.failed"),
            step_id: "",
            timestamp: failedAt,
          };
          flowUpdate.completed_at = failedAt;
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
          updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
            status: "not_completed",
            error: wsEvent.data?.error,
          });
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
        updateFlowData(flowUpdate);
      }
    },
    [selectedFlow, t]
  );

  const socketClient = useWebSocketClient({
    enabled: true,
  });

  useEffect(() => {
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: [["catalog"]],
        include_state: true,
        event_types: CATALOG_EVENT_TYPES,
      },
      handleCatalogEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [handleCatalogEvent, socketClient.subscribe, socketClient.unsubscribe]);

  useEffect(() => {
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: [["cluster"]],
        include_state: true,
        event_types: CLUSTER_EVENT_TYPES,
      },
      handleClusterEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [handleClusterEvent, socketClient.subscribe, socketClient.unsubscribe]);

  useEffect(() => {
    if (visibleFlowIDs.length === 0) {
      return;
    }

    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: visibleFlowIDs.map((flowID) => ["flow", flowID]),
        include_state: false,
        event_types: FLOW_SUMMARY_EVENT_TYPES,
      },
      handleFlowSummaryEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [
    handleFlowSummaryEvent,
    socketClient.subscribe,
    socketClient.unsubscribe,
    visibleFlowIDs,
  ]);

  useEffect(() => {
    if (!selectedFlow) {
      return;
    }

    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: [["flow", selectedFlow]],
        include_state: true,
        event_types: FLOW_EVENT_TYPES,
      },
      handleFlowEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [
    handleFlowEvent,
    selectedFlow,
    socketClient.subscribe,
    socketClient.unsubscribe,
  ]);

  useEffect(() => {
    setEngineSocketStatus(
      socketClient.connectionStatus,
      socketClient.reconnectAttempt
    );
  }, [
    socketClient.connectionStatus,
    socketClient.reconnectAttempt,
    setEngineSocketStatus,
  ]);

  const engineReconnectRef = useRef(engineReconnectRequest);
  useEffect(() => {
    if (engineReconnectRequest === engineReconnectRef.current) {
      return;
    }
    engineReconnectRef.current = engineReconnectRequest;
    socketClient.reconnect();
  }, [engineReconnectRequest, socketClient.reconnect]);

  return <>{children}</>;
};

export default WebSocketProvider;
