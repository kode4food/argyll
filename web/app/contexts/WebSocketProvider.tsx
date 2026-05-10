import React from "react";
import { useWebSocketClient } from "@/app/hooks/useWebSocketClient";
import { useFlowStore } from "@/app/store/flowStore";
import { useCatalogSubscription } from "./useCatalogSubscription";
import { useClusterSubscription } from "./useClusterSubscription";
import { useFlowSummarySubscription } from "./useFlowSummarySubscription";
import { useFlowSubscription } from "./useFlowSubscription";
import { useEngineConnectionSync } from "./useEngineConnectionSync";

const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const selectedFlow = useFlowStore((state) => state.selectedFlow);
  const visibleFlowIDs = useFlowStore((state) => state.visibleFlowIDs);

  const socketClient = useWebSocketClient({ enabled: true });

  useCatalogSubscription(socketClient);
  useClusterSubscription(socketClient);
  useFlowSummarySubscription(socketClient, visibleFlowIDs);
  useFlowSubscription(socketClient, selectedFlow);
  useEngineConnectionSync(socketClient);

  return <>{children}</>;
};

export default WebSocketProvider;
