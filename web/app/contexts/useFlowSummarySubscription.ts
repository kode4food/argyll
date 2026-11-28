import { useCallback, useEffect } from "react";
import { useFlowStore } from "@/app/store/flowStore";
import {
  type FlowStateUpdate,
  toFlowSummaryFromState,
} from "@/app/store/flowStoreHelpers";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import { useT } from "@/app/i18n";
import { META_KEY_PARENT_FLOW_ID } from "@/app/api";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

const FLOW_STARTED_EVENT = "flow_started";

const FLOW_SUMMARY_EVENT_TYPES = [
  FLOW_STARTED_EVENT,
  "flow_completed",
  "flow_failed",
];

const eventTimestamp = (timestamp?: number): string =>
  new Date(timestamp || Date.now()).toISOString();

export function useFlowSummarySubscription(
  socketClient: SocketClient,
  visibleFlowIDs: string[]
) {
  const t = useT();
  const addFlow = useFlowStore((state) => state.addFlow);

  const handleFlowSummaryEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        for (const item of (event as WebSocketSubscribed).items ?? []) {
          const state = item.data as FlowStateUpdate;
          // A lagging replica can still report an active Flow after it settled
          if (state.status === "completed" || state.status === "failed") {
            addFlow(toFlowSummaryFromState(state));
          }
        }
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
          // Child Flows stay out of the list, as they do in flow queries
          if (wsEvent.data?.metadata?.[META_KEY_PARENT_FLOW_ID]) break;
          addFlow({ id: flowId, status: "active", timestamp });
          break;
        case "flow_completed":
          addFlow({ id: flowId, status: "completed", timestamp });
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

  // Flows started anywhere join the list. Once listed, the visible Flow
  // subscription below follows them to completion
  useEffect(() => {
    const subscriptionId = socketClient.subscribe(
      { event_types: [FLOW_STARTED_EVENT] },
      handleFlowSummaryEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [
    handleFlowSummaryEvent,
    socketClient.subscribe,
    socketClient.unsubscribe,
  ]);

  useEffect(() => {
    if (visibleFlowIDs.length === 0) {
      return;
    }

    // Snapshots on reconnect avoid reloading Flow state on every scroll
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: visibleFlowIDs.map((flowID) => ["flow", flowID]),
        include_state: false,
        event_types: FLOW_SUMMARY_EVENT_TYPES,
      },
      handleFlowSummaryEvent,
      { include_state: true }
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
}
