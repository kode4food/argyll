import { useCallback, useRef, RefObject } from "react";
import { WEBSOCKET } from "@/constants/common";

export function useHeartbeat(wsRef: RefObject<WebSocket | null>) {
  const heartbeatIntervalRef = useRef<NodeJS.Timeout | null>(null);

  const startHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
    }
    heartbeatIntervalRef.current = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: "ping" }));
      }
    }, WEBSOCKET.HEARTBEAT_INTERVAL);
  }, [wsRef]);

  const stopHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
  }, []);

  return { startHeartbeat, stopHeartbeat };
}
