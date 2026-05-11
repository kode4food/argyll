import { useCallback, useRef, useState } from "react";
import { WEBSOCKET } from "@/constants/common";
import { ConnectionStatus } from "@/app/types/websocket";

interface UseReconnectOptions {
  enabledRef: React.RefObject<boolean>;
  connectRef: React.RefObject<(() => void) | undefined>;
  onStatusChange: (status: ConnectionStatus) => void;
}

export interface UseReconnectResult {
  reconnectAttempt: number;
  reconnectTimeoutRef: React.RefObject<NodeJS.Timeout | null>;
  reconnectDelayRef: React.RefObject<number>;
  scheduleReconnect: () => void;
  cancelPendingReconnect: () => void;
  resetReconnect: () => void;
}

export function useReconnect({
  enabledRef,
  connectRef,
  onStatusChange,
}: UseReconnectOptions): UseReconnectResult {
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectDelayRef = useRef<number>(WEBSOCKET.INITIAL_RECONNECT_DELAY);

  const cancelPendingReconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  const resetReconnect = useCallback(() => {
    setReconnectAttempt(0);
    reconnectDelayRef.current = WEBSOCKET.INITIAL_RECONNECT_DELAY;
  }, []);

  const applyReconnectAttempt = useCallback(
    (prev: number): number => {
      const nextAttempt = prev + 1;
      if (nextAttempt >= WEBSOCKET.MAX_RECONNECT_ATTEMPTS) {
        onStatusChange("failed");
        return nextAttempt;
      }
      onStatusChange("reconnecting");
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
    },
    [connectRef, onStatusChange]
  );

  const scheduleReconnect = useCallback(() => {
    if (!enabledRef.current) return;
    setReconnectAttempt(applyReconnectAttempt);
  }, [enabledRef, applyReconnectAttempt]);

  return {
    reconnectAttempt,
    reconnectTimeoutRef,
    reconnectDelayRef,
    scheduleReconnect,
    cancelPendingReconnect,
    resetReconnect,
  };
}
