"use client";

import React from "react";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";
import ConnectionStatusIndicator from "./ConnectionStatusIndicator";

const ConnectionStatusWrapper: React.FC = () => {
  const { connectionStatus, reconnectAttempt, reconnect } =
    useWebSocketContext();

  return (
    <ConnectionStatusIndicator
      status={connectionStatus}
      reconnectAttempt={reconnectAttempt}
      onReconnect={reconnect}
    />
  );
};

export default ConnectionStatusWrapper;
