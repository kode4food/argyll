import { useCallback, useEffect } from "react";
import { ExecutionPlan, FlowContext } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import { useT } from "@/app/i18n";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

type FlowStatePayload = {
  id: string;
  status: FlowContext["status"];
  attributes?: Record<string, unknown>;
  plan?: ExecutionPlan;
  executions?: Record<string, unknown>;
  created_at?: string;
  completed_at?: string;
  error?: string;
};

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

type TFn = (key: string) => string;
type UpdateExecution = ReturnType<
  typeof useFlowStore.getState
>["updateExecution"];
type UpdateWorkItem = ReturnType<
  typeof useFlowStore.getState
>["updateWorkItem"];
type AddFlow = ReturnType<typeof useFlowStore.getState>["addFlow"];
type InitializeExecutions = ReturnType<
  typeof useFlowStore.getState
>["initializeExecutions"];

const eventTimestamp = (timestamp?: number): string =>
  new Date(timestamp || Date.now()).toISOString();

const applyStepEvent = (
  wsEvent: WebSocketEvent,
  updateExecution: UpdateExecution
): boolean => {
  const ts = eventTimestamp(wsEvent.timestamp);
  switch (wsEvent.type) {
    case "step_started":
      updateExecution(wsEvent.data?.step_id, {
        status: "active",
        inputs: wsEvent.data?.inputs,
        work_items: wsEvent.data?.work_items || {},
        started_at: ts,
      });
      return true;
    case "step_completed":
      updateExecution(wsEvent.data?.step_id, {
        status: "completed",
        outputs: wsEvent.data?.outputs,
        duration_ms: wsEvent.data?.duration,
        completed_at: ts,
      });
      return true;
    case "step_failed":
      updateExecution(wsEvent.data?.step_id, {
        status: "failed",
        error_message: wsEvent.data?.error,
        completed_at: ts,
      });
      return true;
    case "step_skipped":
      updateExecution(wsEvent.data?.step_id, {
        status: "skipped",
        completed_at: ts,
      });
      return true;
    default:
      return false;
  }
};

const applyWorkItemEvent = (
  wsEvent: WebSocketEvent,
  updateWorkItem: UpdateWorkItem
): boolean => {
  const ts = eventTimestamp(wsEvent.timestamp);
  switch (wsEvent.type) {
    case "work_started":
      updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
        status: "active",
        started_at: ts,
        completed_at: undefined,
        inputs: wsEvent.data?.inputs,
        next_retry_at: undefined,
      });
      return true;
    case "work_succeeded":
      updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
        status: "succeeded",
        completed_at: ts,
        outputs: wsEvent.data?.outputs,
      });
      return true;
    case "work_failed":
      updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
        status: "failed",
        completed_at: ts,
        error: wsEvent.data?.error,
      });
      return true;
    case "work_not_completed":
      updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
        status: "not_completed",
        completed_at: ts,
        error: wsEvent.data?.error,
      });
      return true;
    case "retry_scheduled":
      updateWorkItem(wsEvent.data?.step_id, wsEvent.data?.token, {
        status: "pending",
        retry_count: wsEvent.data?.retry_count ?? 0,
        next_retry_at: wsEvent.data?.next_retry_at,
        error: wsEvent.data?.error,
      });
      return true;
    default:
      return false;
  }
};

interface FlowUpdateContext {
  activeFlowId: string;
  flowData: FlowContext;
  addFlow: AddFlow;
  initializeExecutions: InitializeExecutions;
  t: TFn;
}

const applyFlowUpdate = (
  wsEvent: WebSocketEvent,
  ctx: FlowUpdateContext
): Partial<FlowContext> => {
  const { activeFlowId, flowData, addFlow, initializeExecutions, t } = ctx;
  const flowUpdate: Partial<FlowContext> = {};
  const ts = eventTimestamp(wsEvent.timestamp);

  switch (wsEvent.type) {
    case "flow_started":
      addFlow({ id: activeFlowId, status: "active", timestamp: ts });
      flowUpdate.status = "active";
      flowUpdate.started_at = ts;
      if (wsEvent.data?.plan) {
        initializeExecutions(activeFlowId, wsEvent.data.plan);
      }
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
      addFlow({ id: activeFlowId, status: "completed", timestamp: ts });
      flowUpdate.status = "completed";
      flowUpdate.completed_at = ts;
      break;
    case "flow_failed":
      addFlow({
        id: activeFlowId,
        status: "failed",
        timestamp: ts,
        error: wsEvent.data?.error || t("flow.failed"),
      });
      flowUpdate.status = "failed";
      flowUpdate.error_state = {
        message: wsEvent.data?.error || t("flow.failed"),
        step_id: "",
        timestamp: ts,
      };
      flowUpdate.completed_at = ts;
      break;
    default:
      break;
  }

  return flowUpdate;
};

export function useFlowSubscription(
  socketClient: SocketClient,
  selectedFlow: string | null
) {
  const t = useT();

  const handleFlowEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setFlowNotFound, setFlowState } = useFlowStore.getState();
        const payload = (event as WebSocketSubscribed).items[0]?.data as
          | FlowStatePayload
          | undefined;
        if (!payload) {
          if (selectedFlow) {
            setFlowNotFound(selectedFlow);
          }
          return;
        }
        setFlowState(payload);
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

      if (applyStepEvent(wsEvent, updateExecution)) return;
      if (applyWorkItemEvent(wsEvent, updateWorkItem)) return;

      const flowUpdate = applyFlowUpdate(wsEvent, {
        activeFlowId,
        flowData,
        addFlow,
        initializeExecutions,
        t,
      });
      if (Object.keys(flowUpdate).length > 0) {
        updateFlowData(flowUpdate);
      }
    },
    [selectedFlow, t]
  );

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
}
