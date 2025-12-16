import { useEffect, useRef } from "react";
import { FlowContext } from "@/app/api";
import { WebSocketEvent } from "@/app/hooks/useWebSocketContext";
import {
  createEventKey,
  extractFlowIdFromEvent,
  flowExists,
} from "./flowSelectorUtils";

interface FlowStatusUpdateParams {
  showDropdown: boolean;
  selectedFlow: string | null;
  subscribe: (config: {
    engine_events?: boolean;
    flow_id?: string;
    event_types?: string[];
    from_sequence?: number;
  }) => void;
  events: WebSocketEvent[];
  flows: FlowContext[];
  updateFlowStatus: (
    flowId: string,
    status: FlowContext["status"],
    timestamp?: string
  ) => void;
  loadFlows: () => Promise<void>;
}

export function useFlowStatusUpdates({
  showDropdown,
  selectedFlow,
  subscribe,
  events,
  flows,
  updateFlowStatus,
  loadFlows,
}: FlowStatusUpdateParams) {
  const processedEventsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (showDropdown || !selectedFlow) {
      subscribe({
        event_types: ["flow_started", "flow_completed", "flow_failed"],
      });
    } else {
      subscribe({
        event_types: [],
      });
    }
  }, [showDropdown, selectedFlow, subscribe]);

  useEffect(() => {
    const latestEvent = events[events.length - 1];
    if (!latestEvent) return;

    const eventKey = createEventKey(latestEvent.id, latestEvent.sequence);
    if (processedEventsRef.current.has(eventKey)) {
      return;
    }

    processedEventsRef.current.add(eventKey);

    const flowId = extractFlowIdFromEvent(latestEvent.id);
    if (!flowId) return;

    const eventType = latestEvent.type;

    if (eventType === "flow_started") {
      if (flowExists(flows, flowId)) {
        updateFlowStatus(flowId, "active");
      } else {
        loadFlows();
      }
    } else if (eventType === "flow_completed") {
      updateFlowStatus(
        flowId,
        "completed",
        new Date(latestEvent.timestamp).toISOString()
      );
    } else if (eventType === "flow_failed") {
      updateFlowStatus(
        flowId,
        "failed",
        new Date(latestEvent.timestamp).toISOString()
      );
    }
  }, [events, flows, updateFlowStatus, loadFlows]);
}
