import { useCallback, useEffect } from "react";
import { useFlowStore, StepHealthInfo } from "@/app/store/flowStore";
import { WebSocketEvent, WebSocketSubscribed } from "@/app/types/websocket";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type ClusterNode = { health?: Record<string, StepHealthInfo> };
type ClusterPayload = { nodes?: Record<string, ClusterNode> };

type SocketClient = ReturnType<typeof useWebSocketClient>;

const CLUSTER_EVENT_TYPES = ["step_health_changed"];

export function useClusterSubscription(socketClient: SocketClient) {
  const updateStepHealth = useFlowStore((state) => state.updateStepHealth);

  const handleClusterEvent = useCallback(
    (event: WebSocketEvent | WebSocketSubscribed) => {
      if (event.type === "subscribed") {
        const { setHealthState } = useFlowStore.getState();
        const payload = (event as WebSocketSubscribed).items[0]?.data as
          | ClusterPayload
          | undefined;
        const nodes = payload?.nodes ?? {};
        const healthByNode: Record<string, Record<string, StepHealthInfo>> = {};
        for (const [nodeId, node] of Object.entries(nodes)) {
          if (node.health) {
            healthByNode[nodeId] = node.health;
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
}
