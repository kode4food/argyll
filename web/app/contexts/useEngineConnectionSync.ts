import { useEffect, useRef } from "react";
import { useFlowStore } from "@/app/store/flowStore";
import type { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

type SocketClient = ReturnType<typeof useWebSocketClient>;

export function useEngineConnectionSync(socketClient: SocketClient) {
  const setEngineSocketStatus = useFlowStore(
    (state) => state.setEngineSocketStatus
  );
  const engineReconnectRequest = useFlowStore(
    (state) => state.engineReconnectRequest
  );

  useEffect(() => {
    setEngineSocketStatus(
      socketClient.connectionStatus,
      socketClient.reconnectAttempt
    );
  }, [
    socketClient.connectionStatus,
    socketClient.reconnectAttempt,
    setEngineSocketStatus,
  ]);

  const engineReconnectRef = useRef(engineReconnectRequest);
  useEffect(() => {
    if (engineReconnectRequest === engineReconnectRef.current) {
      return;
    }
    engineReconnectRef.current = engineReconnectRequest;
    socketClient.reconnect();
  }, [engineReconnectRequest, socketClient.reconnect]);
}
