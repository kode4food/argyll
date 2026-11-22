import { useCallback, useEffect, useRef, useState } from "react";
import { API_CONFIG } from "@/constants/common";
import {
  ConnectionStatus,
  WebSocketEvent,
  WebSocketSubscribed,
} from "@/app/types/websocket";
import { useHeartbeat } from "./useHeartbeat";
import { useReconnect } from "./useReconnect";
import { useSubscriptions } from "./useSubscriptions";
import { useWebSocketHandlers } from "./useWebSocketHandlers";

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
  const wsRef = useRef<WebSocket | null>(null);
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

  const {
    reconnectAttempt,
    reconnectTimeoutRef,
    scheduleReconnect,
    cancelPendingReconnect,
    resetReconnect,
  } = useReconnect({
    enabledRef,
    connectRef,
    onStatusChange: setConnectionStatus,
  });

  const teardown = useCallback(() => {
    cancelPendingReconnect();
    stopHeartbeat();
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [cancelPendingReconnect, stopHeartbeat]);

  const { handleOpen, handleClose, handleError, handleMessage } =
    useWebSocketHandlers({
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
    });

  const openWebSocket = useCallback(
    (handlers: {
      onopen: () => void;
      onclose: (e: CloseEvent) => void;
      onerror: () => void;
      onmessage: (e: MessageEvent) => void;
    }) => {
      try {
        const ws = new WebSocket(API_CONFIG.WS_URL);
        wsRef.current = ws;
        ws.onopen = handlers.onopen;
        ws.onclose = handlers.onclose;
        ws.onerror = handlers.onerror;
        ws.onmessage = handlers.onmessage;
      } catch (error) {
        console.error("Failed to create WebSocket:", error);
        if (!reconnectTimeoutRef.current) {
          scheduleReconnect();
        }
      }
    },
    [reconnectTimeoutRef, scheduleReconnect]
  );

  const connect = useCallback(() => {
    if (!enabledRef.current) return;

    cancelPendingReconnect();

    const ws = wsRef.current;
    if (
      ws?.readyState === WebSocket.CONNECTING ||
      ws?.readyState === WebSocket.OPEN
    ) {
      return;
    }
    ws?.close();
    wsRef.current = null;

    stopHeartbeat();
    setConnectionStatus("connecting");
    openWebSocket({
      onopen: handleOpen,
      onclose: handleClose,
      onerror: handleError,
      onmessage: handleMessage,
    });
  }, [
    cancelPendingReconnect,
    handleClose,
    handleError,
    handleMessage,
    handleOpen,
    openWebSocket,
    stopHeartbeat,
  ]);

  connectRef.current = connect;

  const reconnect = useCallback(() => {
    resetReconnect();
    teardown();
    connect();
  }, [connect, resetReconnect, teardown]);

  useEffect(() => {
    if (!enabled) {
      teardown();
      setConnectionStatus("disconnected");
    } else {
      connect();
    }
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
