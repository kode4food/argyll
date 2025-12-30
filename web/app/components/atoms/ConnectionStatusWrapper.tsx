"use client";

import React from "react";
import {
  useEngineConnectionStatus,
  useEngineReconnectAttempt,
  useRequestEngineReconnect,
} from "../../store/flowStore";
import ConnectionStatusIndicator from "./ConnectionStatusIndicator";

const ConnectionStatusWrapper: React.FC = () => {
  const connectionStatus = useEngineConnectionStatus();
  const reconnectAttempt = useEngineReconnectAttempt();
  const reconnect = useRequestEngineReconnect();

  return (
    <ConnectionStatusIndicator
      status={connectionStatus}
      reconnectAttempt={reconnectAttempt}
      onReconnect={reconnect}
    />
  );
};

export default ConnectionStatusWrapper;
