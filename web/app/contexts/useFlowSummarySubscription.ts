import { useCallback, useEffect } from "react";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import { useT } from "@/app/i18n";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

const FLOW_SUMMARY_EVENT_TYPES = [
  "flow_started",
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
}
