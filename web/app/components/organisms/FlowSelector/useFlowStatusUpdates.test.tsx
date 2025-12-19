import { act, renderHook } from "@testing-library/react";
import { FlowContext } from "@/app/api";
import { WebSocketEvent } from "@/app/hooks/useWebSocketContext";
import { useFlowStatusUpdates } from "./useFlowStatusUpdates";

const baseFlow: FlowContext = {
  id: "flow-1",
  status: "pending",
  state: {},
  started_at: "2024-01-01T00:00:00Z",
};

const makeEvent = (
  type: string,
  sequence: number,
  timestamp: number,
  flowId = "flow-1"
): WebSocketEvent => ({
  id: ["flow", flowId],
  sequence,
  type,
  timestamp,
  data: {},
});

describe("useFlowStatusUpdates", () => {
  it("subscribes per dropdown state", () => {
    const subscribe = jest.fn();
    const props = {
      showDropdown: true,
      selectedFlow: null as string | null,
      subscribe,
      events: [] as WebSocketEvent[],
      flows: [baseFlow],
      updateFlowStatus: jest.fn(),
      loadFlows: jest.fn(),
    };

    const { rerender } = renderHook(
      (hookProps) => useFlowStatusUpdates(hookProps),
      { initialProps: props }
    );

    expect(subscribe).toHaveBeenCalledWith({
      event_types: ["flow_started", "flow_completed", "flow_failed"],
    });

    act(() => {
      rerender({
        ...props,
        showDropdown: false,
        selectedFlow: "flow-1",
      });
    });

    expect(subscribe).toHaveBeenLastCalledWith({ event_types: [] });
  });

  it("processes flow events once", () => {
    const updateFlowStatus = jest.fn();
    const props = {
      showDropdown: false,
      selectedFlow: "flow-1",
      subscribe: jest.fn(),
      events: [] as WebSocketEvent[],
      flows: [baseFlow],
      updateFlowStatus,
      loadFlows: jest.fn(),
    };

    const started = makeEvent("flow_started", 1, 1_730_000_000_000);
    const completed = makeEvent("flow_completed", 2, 1_730_000_001_000);
    const failed = makeEvent("flow_failed", 3, 1_730_000_002_000);

    const { rerender } = renderHook(
      (hookProps) => useFlowStatusUpdates(hookProps),
      { initialProps: props }
    );

    act(() => {
      rerender({ ...props, events: [started] });
    });
    expect(updateFlowStatus).toHaveBeenCalledWith("flow-1", "active");

    act(() => {
      rerender({ ...props, events: [started, completed] });
    });
    expect(updateFlowStatus).toHaveBeenCalledWith(
      "flow-1",
      "completed",
      new Date(completed.timestamp).toISOString()
    );

    act(() => {
      rerender({ ...props, events: [started, completed, failed] });
    });
    expect(updateFlowStatus).toHaveBeenCalledWith(
      "flow-1",
      "failed",
      new Date(failed.timestamp).toISOString()
    );

    act(() => {
      rerender({ ...props, events: [started, completed, failed] });
    });
    expect(updateFlowStatus).toHaveBeenCalledTimes(3);
  });

  it("loads flows for unknown start event", () => {
    const updateFlowStatus = jest.fn();
    const loadFlows = jest.fn();
    const startEvent = makeEvent("flow_started", 1, 1_730_000_000_000, "new");

    const props = {
      showDropdown: false,
      selectedFlow: null as string | null,
      subscribe: jest.fn(),
      events: [] as WebSocketEvent[],
      flows: [],
      updateFlowStatus,
      loadFlows,
    };

    const { rerender } = renderHook(
      (hookProps) => useFlowStatusUpdates(hookProps),
      { initialProps: props }
    );

    act(() => {
      rerender({ ...props, events: [startEvent] });
    });

    expect(loadFlows).toHaveBeenCalledTimes(1);
    expect(updateFlowStatus).not.toHaveBeenCalled();
  });

  it("ignores events with malformed IDs", () => {
    const updateFlowStatus = jest.fn();
    const loadFlows = jest.fn();
    const malformedEvent: WebSocketEvent = {
      id: ["invalid"],
      sequence: 1,
      type: "flow_started",
      timestamp: 1_730_000_000_000,
      data: {},
    };

    const props = {
      showDropdown: false,
      selectedFlow: null as string | null,
      subscribe: jest.fn(),
      events: [] as WebSocketEvent[],
      flows: [baseFlow],
      updateFlowStatus,
      loadFlows,
    };

    const { rerender } = renderHook(
      (hookProps) => useFlowStatusUpdates(hookProps),
      { initialProps: props }
    );

    act(() => {
      rerender({ ...props, events: [malformedEvent] });
    });

    expect(updateFlowStatus).not.toHaveBeenCalled();
    expect(loadFlows).not.toHaveBeenCalled();
  });

  it("ignores unhandled event types", () => {
    const updateFlowStatus = jest.fn();
    const unknownEvent = makeEvent("unknown_type", 1, 1_730_000_000_000);

    const props = {
      showDropdown: false,
      selectedFlow: null as string | null,
      subscribe: jest.fn(),
      events: [] as WebSocketEvent[],
      flows: [baseFlow],
      updateFlowStatus,
      loadFlows: jest.fn(),
    };

    const { rerender } = renderHook(
      (hookProps) => useFlowStatusUpdates(hookProps),
      { initialProps: props }
    );

    act(() => {
      rerender({ ...props, events: [unknownEvent] });
    });

    expect(updateFlowStatus).not.toHaveBeenCalled();
  });
});
