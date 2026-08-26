import { useCallback, useEffect } from "react";
import { Space, Step } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type CatalogPayload = {
  steps?: Record<string, Step>;
  spaces?: Record<string, Space>;
  selection?: Record<string, string[]>;
};

type SocketClient = ReturnType<typeof useWebSocketClient>;

const catalogEventTypes = [
  "step_registered",
  "step_unregistered",
  "step_updated",
  "space_registered",
  "space_unregistered",
  "space_updated",
];

export function useCatalogSubscription(socketClient: SocketClient) {
  const addStep = useFlowStore((state) => state.addStep);
  const updateStep = useFlowStore((state) => state.updateStep);
  const removeStep = useFlowStore((state) => state.removeStep);
  const setStepSpaces = useFlowStore((state) => state.setStepSpaces);
  const setSpace = useFlowStore((state) => state.setSpace);
  const removeSpace = useFlowStore((state) => state.removeSpace);

  const handleCatalogEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setCatalogState } = useFlowStore.getState();
        const payload = (event as WebSocketSubscribed).items[0]?.data as
          | CatalogPayload
          | undefined;
        setCatalogState(
          payload?.steps ?? {},
          payload?.spaces ?? {},
          payload?.selection ?? {}
        );
        return;
      }

      const wsEvent = event as WebSocketEvent;
      switch (wsEvent.type) {
        case "step_registered": {
          const step = wsEvent.data?.step;
          if (step) {
            addStep(step);
            setStepSpaces(step.id, wsEvent.data?.spaces ?? []);
          }
          break;
        }
        case "step_unregistered": {
          const stepId = wsEvent.data?.step_id;
          if (stepId) {
            removeStep(stepId);
            setStepSpaces(stepId, []);
          }
          break;
        }
        case "step_updated": {
          const step = wsEvent.data?.step;
          if (step) {
            updateStep(step);
            setStepSpaces(step.id, wsEvent.data?.spaces ?? []);
          }
          break;
        }
        case "space_registered":
        case "space_updated": {
          const space = wsEvent.data?.space;
          if (space) {
            setSpace(space, Object.keys(wsEvent.data?.steps ?? {}));
          }
          break;
        }
        case "space_unregistered": {
          const spaceId = wsEvent.data?.space_id;
          if (spaceId) removeSpace(spaceId);
          break;
        }
        default:
          break;
      }
    },
    [addStep, removeStep, updateStep, setStepSpaces]
  );

  useEffect(() => {
    const subscriptionId = socketClient.subscribe(
      {
        aggregate_ids: [["catalog"]],
        include_state: true,
        event_types: catalogEventTypes,
      },
      handleCatalogEvent
    );
    return () => {
      socketClient.unsubscribe(subscriptionId);
    };
  }, [handleCatalogEvent, socketClient.subscribe, socketClient.unsubscribe]);
}
