"use client";

import React, {
  createContext,
  ReactNode,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { API_CONFIG, WEBSOCKET } from "@/constants/common";

export interface WebSocketEvent {
  type: string;
  data: any;
  timestamp: number;
  sequence: number;
  id: string[];
}

export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "reconnecting"
  | "failed";

interface WebSocketContextType {
  isConnected: boolean;
  connectionStatus: ConnectionStatus;
  events: WebSocketEvent[];
  reconnectAttempt: number;
  subscribe: (subscription: {
    engine_events?: boolean;
    flow_id?: string; // Empty string = no flow events
    event_types?: string[]; // Filter for specific event types
    from_sequence?: number; // Start replay from this sequence
  }) => void;
  reconnect: () => void;
  registerConsumer: () => string;
  unregisterConsumer: (consumerId: string) => void;
  updateConsumerCursor: (consumerId: string, cursor: number) => void;
}

const WebSocketContext = createContext<WebSocketContextType | null>(null);

export const WebSocketProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] =
    useState<ConnectionStatus>("connecting");
  const [events, setEvents] = useState<WebSocketEvent[]>([]);
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectDelayRef = useRef<number>(WEBSOCKET.INITIAL_RECONNECT_DELAY);
  const heartbeatIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const connectRef = useRef<(() => void) | undefined>(undefined);

  const consumerCursorsRef = useRef<Map<string, number>>(new Map());
  const nextConsumerIdRef = useRef(0);

  const currentSubscriptionRef = useRef<{
    engine_events?: boolean;
    flow_id?: string;
    event_types?: string[];
    from_sequence?: number;
  }>({});

  const [pendingSubscription, setPendingSubscription] = useState<{
    engine_events?: boolean;
    flow_id?: string;
    event_types?: string[];
    from_sequence?: number;
  } | null>(null);

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

  const scheduleReconnect = useCallback(() => {
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
        setIsConnected(true);
        setConnectionStatus("connected");
        setReconnectAttempt(0);
        reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;

        startHeartbeat();

        if (pendingSubscription) {
          ws.send(
            JSON.stringify({
              type: "subscribe",
              data: pendingSubscription,
            })
          );
          setPendingSubscription(null);
        }
      };

      ws.onclose = (event) => {
        setIsConnected(false);
        stopHeartbeat();
        wsRef.current = null;

        if (event.code !== 1000 && !reconnectTimeoutRef.current) {
          scheduleReconnect();
        } else {
          setConnectionStatus("disconnected");
        }
      };

      ws.onerror = () => {
        setIsConnected(false);
        setConnectionStatus("disconnected");
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === "pong") {
            return;
          }
          setEvents((prev) => {
            const updated = [...prev, data];

            if (consumerCursorsRef.current.size > 0) {
              const minCursor = Math.min(
                ...consumerCursorsRef.current.values()
              );
              for (let i = 0; i < minCursor; i++) {
                delete updated[i];
              }
            }

            return updated;
          });
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
  }, [pendingSubscription, startHeartbeat, stopHeartbeat, scheduleReconnect]);

  connectRef.current = connect;

  const subscribe = useCallback(
    (subscription: {
      engine_events?: boolean;
      flow_id?: string;
      event_types?: string[];
      from_sequence?: number;
    }) => {
      const current = currentSubscriptionRef.current;

      const newSubscription = {
        engine_events:
          subscription.engine_events !== undefined
            ? subscription.engine_events
            : current.engine_events,
        flow_id:
          subscription.flow_id !== undefined
            ? subscription.flow_id
            : current.flow_id,
        event_types:
          subscription.event_types !== undefined
            ? subscription.event_types
            : current.event_types,
        from_sequence:
          subscription.from_sequence !== undefined
            ? subscription.from_sequence
            : current.from_sequence,
      };

      currentSubscriptionRef.current = newSubscription;

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(
          JSON.stringify({
            type: "subscribe",
            data: newSubscription,
          })
        );
      } else {
        setPendingSubscription(newSubscription);
      }
    },
    []
  );

  const registerConsumer = useCallback(() => {
    const consumerId = `consumer-${nextConsumerIdRef.current++}`;
    consumerCursorsRef.current.set(consumerId, 0);
    return consumerId;
  }, []);

  const unregisterConsumer = useCallback((consumerId: string) => {
    consumerCursorsRef.current.delete(consumerId);
  }, []);

  const updateConsumerCursor = useCallback(
    (consumerId: string, cursor: number) => {
      consumerCursorsRef.current.set(consumerId, cursor);
    },
    []
  );

  const manualReconnect = useCallback(() => {
    setReconnectAttempt(0);
    reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;
    connect();
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  return (
    <WebSocketContext.Provider
      value={{
        isConnected,
        connectionStatus,
        events,
        reconnectAttempt,
        subscribe,
        reconnect: manualReconnect,
        registerConsumer,
        unregisterConsumer,
        updateConsumerCursor,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
};

export const useWebSocketContext = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error(
      "useWebSocketContext must be used within a WebSocketProvider"
    );
  }
  return context;
};
