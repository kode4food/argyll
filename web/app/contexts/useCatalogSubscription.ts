import { useCallback, useEffect } from "react";
import { Step } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type CatalogPayload = { steps?: Record<string, Step> };

type SocketClient = ReturnType<typeof useWebSocketClient>;

const CATALOG_EVENT_TYPES = [
  "step_registered",
  "step_unregistered",
  "step_updated",
];

export function useCatalogSubscription(socketClient: SocketClient) {
  const addStep = useFlowStore((state) => state.addStep);
  const updateStep = useFlowStore((state) => state.updateStep);
  const removeStep = useFlowStore((state) => state.removeStep);

  const handleCatalogEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setCatalogState } = useFlowStore.getState();
        const payload = (event as WebSocketSubscribed).items[0]?.data as
          | CatalogPayload
          | undefined;
        setCatalogState(payload?.steps ?? {});
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
}
