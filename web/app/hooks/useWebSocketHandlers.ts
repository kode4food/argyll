import { useCallback } from "react";
import {
  ConnectionStatus,
  WebSocketEvent,
  WebSocketSubscribed,
} from "@/app/types/websocket";
import { sendSubscribeMessage, SubscriptionEntry } from "./useSubscriptions";

interface UseWebSocketHandlersOptions {
  wsRef: React.RefObject<WebSocket | null>;
  enabledRef: React.RefObject<boolean>;
  onEventRef: React.RefObject<
    ((event: WebSocketEvent | WebSocketSubscribed) => void) | undefined
  >;
  subscriptionsRef: React.RefObject<Map<string, SubscriptionEntry>>;
  reconnectTimeoutRef: React.RefObject<NodeJS.Timeout | null>;
  startHeartbeat: () => void;
  stopHeartbeat: () => void;
  scheduleReconnect: () => void;
  resetReconnect: () => void;
  setConnectionStatus: (status: ConnectionStatus) => void;
}

export function useWebSocketHandlers({
  wsRef,
  enabledRef,
  onEventRef,
  subscriptionsRef,
  reconnectTimeoutRef,
  startHeartbeat,
  stopHeartbeat,
  scheduleReconnect,
  resetReconnect,
  setConnectionStatus,
}: UseWebSocketHandlersOptions) {
  const handleOpen = useCallback(() => {
    setConnectionStatus("connected");
    resetReconnect();
    startHeartbeat();
    if (wsRef.current) {
      for (const { subscription } of subscriptionsRef.current.values()) {
        sendSubscribeMessage(wsRef.current, subscription);
      }
    }
  }, [
    resetReconnect,
    startHeartbeat,
    subscriptionsRef,
    wsRef,
    setConnectionStatus,
  ]);

  const handleClose = useCallback(
    (event: CloseEvent) => {
      stopHeartbeat();
      wsRef.current = null;
      if (
        !enabledRef.current ||
        event.code === 1000 ||
        reconnectTimeoutRef.current
      ) {
        setConnectionStatus("disconnected");
      } else {
        scheduleReconnect();
      }
    },
    [
      enabledRef,
      reconnectTimeoutRef,
      scheduleReconnect,
      setConnectionStatus,
      stopHeartbeat,
      wsRef,
    ]
  );

  const handleError = useCallback(() => {
    setConnectionStatus("disconnected");
  }, [setConnectionStatus]);

  const routeWebSocketMessage = useCallback(
    (data: WebSocketEvent | WebSocketSubscribed) => {
      const sub = data.sub_id
        ? subscriptionsRef.current.get(data.sub_id)
        : undefined;
      if (sub?.onEvent) {
        sub.onEvent(data);
      } else {
        onEventRef.current?.(data);
      }
    },
    [onEventRef, subscriptionsRef]
  );

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data) as
          | WebSocketEvent
          | WebSocketSubscribed
          | { type: "pong" };
        if (data.type !== "pong") {
          routeWebSocketMessage(data as WebSocketEvent | WebSocketSubscribed);
        }
      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    },
    [routeWebSocketMessage]
  );

  return { handleOpen, handleClose, handleError, handleMessage };
}
