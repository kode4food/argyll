import { useCallback, useEffect, useRef, useState } from "react";
import { API_CONFIG, WEBSOCKET } from "@/constants/common";
import {
  ConnectionStatus,
  WebSocketEvent,
  WebSocketSubscribe,
  WebSocketSubscribed,
} from "@/app/types/websocket";
import { useHeartbeat } from "./useHeartbeat";
import { useSubscriptions, sendSubscribeMessage } from "./useSubscriptions";

interface UseWebSocketClientOptions {
  enabled?: boolean;
  onEvent?: (event: WebSocketEvent | WebSocketSubscribed) => void;
}

export const useWebSocketClient = ({
  enabled = true,
  onEvent,
}: UseWebSocketClientOptions) => {
  const [connectionStatus, setConnectionStatus] =
    useState<ConnectionStatus>("connecting");
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectDelayRef = useRef<number>(WEBSOCKET.INITIAL_RECONNECT_DELAY);
  const connectRef = useRef<(() => void) | undefined>(undefined);
  const enabledRef = useRef(enabled);
  const onEventRef = useRef(onEvent);

  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  const { startHeartbeat, stopHeartbeat } = useHeartbeat(wsRef);
  const { subscribe, unsubscribe, subscriptionsRef } = useSubscriptions(wsRef);

  const teardown = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    stopHeartbeat();
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [stopHeartbeat]);

  const scheduleReconnect = useCallback(() => {
    if (!enabledRef.current) return;

    setReconnectAttempt((prev) => {
      const nextAttempt = prev + 1;
      if (nextAttempt >= WEBSOCKET.MAX_RECONNECT_ATTEMPTS) {
        setConnectionStatus("failed");
        return nextAttempt;
      }
      setConnectionStatus("reconnecting");
      const delay = Math.min(
        reconnectDelayRef.current,
        WEBSOCKET.MAX_RECONNECT_DELAY
      );
      reconnectTimeoutRef.current = setTimeout(() => {
        reconnectTimeoutRef.current = null;
        reconnectDelayRef.current = Math.min(
          reconnectDelayRef.current * WEBSOCKET.RECONNECT_MULTIPLIER,
          WEBSOCKET.MAX_RECONNECT_DELAY
        );
        connectRef.current?.();
      }, delay);
      return nextAttempt;
    });
  }, []);

  const handleOpen = useCallback(() => {
    setConnectionStatus("connected");
    setReconnectAttempt(0);
    reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;
    startHeartbeat();
    if (wsRef.current) {
      for (const { subscription } of subscriptionsRef.current.values()) {
        sendSubscribeMessage(wsRef.current, subscription);
      }
    }
  }, [startHeartbeat, subscriptionsRef]);

  const handleClose = useCallback(
    (event: CloseEvent) => {
      stopHeartbeat();
      wsRef.current = null;
      if (!enabledRef.current) {
        setConnectionStatus("disconnected");
        return;
      }
      if (event.code !== 1000 && !reconnectTimeoutRef.current) {
        scheduleReconnect();
      } else {
        setConnectionStatus("disconnected");
      }
    },
    [scheduleReconnect, stopHeartbeat]
  );

  const handleError = useCallback(() => {
    setConnectionStatus("disconnected");
  }, []);

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data) as
          | WebSocketEvent
          | WebSocketSubscribed
          | { type: "pong" };
        if (data.type === "pong") return;

        const routedEvent = data as WebSocketEvent | WebSocketSubscribed;
        const subscriptionId = routedEvent.sub_id;
        if (subscriptionId) {
          const sub = subscriptionsRef.current.get(subscriptionId);
          if (sub?.onEvent) {
            sub.onEvent(routedEvent);
            return;
          }
        }
        onEventRef.current?.(routedEvent);
      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    },
    [subscriptionsRef]
  );

  const connect = useCallback(() => {
    if (!enabledRef.current) return;

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (
      wsRef.current &&
      (wsRef.current.readyState === WebSocket.CONNECTING ||
        wsRef.current.readyState === WebSocket.OPEN)
    ) {
      return;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    stopHeartbeat();
    setConnectionStatus("connecting");

    try {
      const ws = new WebSocket(API_CONFIG.WS_URL);
      wsRef.current = ws;
      ws.onopen = handleOpen;
      ws.onclose = handleClose;
      ws.onerror = handleError;
      ws.onmessage = handleMessage;
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
      if (!reconnectTimeoutRef.current) {
        scheduleReconnect();
      }
    }
  }, [
    handleClose,
    handleError,
    handleMessage,
    handleOpen,
    scheduleReconnect,
    stopHeartbeat,
  ]);

  connectRef.current = connect;

  const reconnect = useCallback(() => {
    setReconnectAttempt(0);
    reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;
    teardown();
    connect();
  }, [connect, teardown]);

  useEffect(() => {
    if (!enabled) {
      teardown();
      setConnectionStatus("disconnected");
      return;
    }
    connect();
    return () => {
      teardown();
    };
  }, [connect, enabled, teardown]);

  return {
    connectionStatus,
    reconnectAttempt,
    subscribe,
    unsubscribe,
    reconnect,
  };
};
