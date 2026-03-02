import { useCallback, useEffect, useRef, useState } from "react";
import { API_CONFIG, WEBSOCKET } from "@/constants/common";
import {
  ConnectionStatus,
  WebSocketEvent,
  WebSocketSubscribe,
  WebSocketSubscribed,
  WebSocketUnsubscribe,
} from "@/app/types/websocket";

interface UseWebSocketClientOptions {
  enabled?: boolean;
  onEvent?: (event: WebSocketEvent | WebSocketSubscribed) => void;
}

interface SubscriptionEntry {
  onEvent?: (event: WebSocketEvent | WebSocketSubscribed) => void;
  subscription: WebSocketSubscribe;
}

const sendSubscribeMessage = (
  ws: WebSocket,
  subscription: WebSocketSubscribe
) => {
  ws.send(
    JSON.stringify({
      type: "subscribe",
      data: subscription,
    })
  );
};

const sendUnsubscribeMessage = (
  ws: WebSocket,
  subscription: WebSocketUnsubscribe
) => {
  ws.send(
    JSON.stringify({
      type: "unsubscribe",
      data: subscription,
    })
  );
};

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
  const heartbeatIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const connectRef = useRef<(() => void) | undefined>(undefined);
  const enabledRef = useRef(enabled);
  const onEventRef = useRef(onEvent);
  const nextSubscriptionIdRef = useRef(0);
  const subscriptionsRef = useRef<Map<string, SubscriptionEntry>>(new Map());

  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  const startHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
    }

    heartbeatIntervalRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: "ping" }));
      }
    }, WEBSOCKET.HEARTBEAT_INTERVAL);
  }, []);

  const stopHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
  }, []);

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
    if (!enabledRef.current) {
      return;
    }

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

  const connect = useCallback(() => {
    if (!enabledRef.current) {
      return;
    }

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

      ws.onopen = () => {
        setConnectionStatus("connected");
        setReconnectAttempt(0);
        reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;
        startHeartbeat();

        for (const { subscription } of subscriptionsRef.current.values()) {
          sendSubscribeMessage(ws, subscription);
        }
      };

      ws.onclose = (event) => {
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
      };

      ws.onerror = () => {
        setConnectionStatus("disconnected");
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as
            | WebSocketEvent
            | WebSocketSubscribed
            | { type: "pong" };
          if (data.type === "pong") {
            return;
          }

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
      };
    } catch (error) {
      console.error("Failed to create WebSocket:", error);
      if (!reconnectTimeoutRef.current) {
        scheduleReconnect();
      }
    }
  }, [scheduleReconnect, startHeartbeat, stopHeartbeat]);

  connectRef.current = connect;

  const subscribe = useCallback(
    (
      subscription: WebSocketSubscribe,
      handler?: (event: WebSocketEvent | WebSocketSubscribed) => void
    ) => {
      const subscriptionId = String(nextSubscriptionIdRef.current);
      nextSubscriptionIdRef.current += 1;

      const nextSubscription = {
        ...subscription,
        sub_id: subscriptionId,
      };

      subscriptionsRef.current.set(subscriptionId, {
        subscription: nextSubscription,
        onEvent: handler,
      });

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        sendSubscribeMessage(wsRef.current, nextSubscription);
      }

      return subscriptionId;
    },
    []
  );

  const unsubscribe = useCallback((subscriptionId: string) => {
    const hadSubscription = subscriptionsRef.current.delete(subscriptionId);
    if (!hadSubscription || wsRef.current?.readyState !== WebSocket.OPEN) {
      return;
    }

    sendUnsubscribeMessage(wsRef.current, {
      sub_id: subscriptionId,
    });
  }, []);

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
