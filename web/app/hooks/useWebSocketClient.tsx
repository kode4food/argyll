import { useCallback, useEffect, useRef, useState } from "react";
import { API_CONFIG, WEBSOCKET } from "@/constants/common";
import { useAuthStore } from "@/app/store/authStore";
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
      connectRef,
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
        // Browsers cannot set headers here, so a token rides as a subprotocol
        const { token } = useAuthStore.getState();
        const ws = token
          ? new WebSocket(API_CONFIG.WS_URL, [
              WEBSOCKET.PROTOCOL,
              WEBSOCKET.TOKEN_PREFIX + token,
            ])
          : new WebSocket(API_CONFIG.WS_URL);
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

    // The server refuses an expired token at the handshake, which a browser
    // reports only as a dropped connection
    const { isEnabled, expiresAt, login } = useAuthStore.getState();
    if (isEnabled && expiresAt !== null && expiresAt <= Date.now()) {
      void login().catch((error: unknown) =>
        console.error("Sign-in failed", error)
      );
      return;
    }

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
    setConnectionStatus((prev) => (prev === "connected" ? prev : "connecting"));
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
