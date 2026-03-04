import { renderHook, act } from "@testing-library/react";
import { useWebSocketClient } from "./useWebSocketClient";
import { WEBSOCKET } from "@/constants/common";

type MockWebSocketInstance = {
  url: string;
  readyState: number;
  send: jest.Mock;
  close: jest.Mock;
  onopen: (() => void) | null;
  onclose: ((event: { code: number }) => void) | null;
  onmessage: ((event: { data: string }) => void) | null;
  onerror: (() => void) | null;
  triggerOpen: () => void;
  triggerClose: (code?: number) => void;
  triggerMessage: (data: unknown) => void;
  triggerError: () => void;
};

const instances: MockWebSocketInstance[] = [];

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState = MockWebSocket.CONNECTING;
  send = jest.fn();
  close = jest.fn(() => {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code: 1000 });
  });
  onopen: (() => void) | null = null;
  onclose: ((event: { code: number }) => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    instances.push(this as unknown as MockWebSocketInstance);
  }

  triggerOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  triggerClose(code = 1006) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code });
  }

  triggerMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  triggerError() {
    this.onerror?.();
  }
}

describe("useWebSocketClient", () => {
  beforeEach(() => {
    instances.length = 0;
    // @ts-expect-error - test shim
    global.WebSocket = MockWebSocket;
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.clearAllMocks();
    jest.useRealTimers();
  });

  test("connects when enabled and sends pending subscriptions on open", () => {
    const onEvent = jest.fn();
    const handler = jest.fn();
    const { result } = renderHook(() =>
      useWebSocketClient({ enabled: true, onEvent })
    );

    expect(result.current.connectionStatus).toBe("connecting");

    let subscriptionId = "";
    act(() => {
      subscriptionId = result.current.subscribe(
        { aggregate_ids: [["catalog"]] },
        handler
      );
    });

    expect(subscriptionId).toBe("0");
    expect(instances).toHaveLength(1);

    act(() => {
      instances[0].triggerOpen();
    });

    expect(result.current.connectionStatus).toBe("connected");
    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: { aggregate_ids: [["catalog"]], sub_id: "0" },
      })
    );
  });

  test("sends subscription immediately when socket is open", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
    });

    act(() => {
      result.current.subscribe({ aggregate_ids: [["flow", "flow-1"]] });
    });

    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: { aggregate_ids: [["flow", "flow-1"]], sub_id: "0" },
      })
    );
  });

  test("sends include_state when requested", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
    });

    act(() => {
      result.current.subscribe({
        aggregate_ids: [["catalog"]],
        include_state: true,
      });
    });

    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: {
          aggregate_ids: [["catalog"]],
          include_state: true,
          sub_id: "0",
        },
      })
    );
  });

  test("sends multi-aggregate subscriptions", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
    });

    act(() => {
      result.current.subscribe({
        aggregate_ids: [
          ["flow", "flow-1"],
          ["flow", "flow-2"],
        ],
      });
    });

    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: {
          aggregate_ids: [
            ["flow", "flow-1"],
            ["flow", "flow-2"],
          ],
          sub_id: "0",
        },
      })
    );
  });

  test("routes messages to the matching subscription handler", () => {
    const fallbackHandler = jest.fn();
    const subscriptionHandler = jest.fn();
    const { result } = renderHook(() =>
      useWebSocketClient({ enabled: true, onEvent: fallbackHandler })
    );

    act(() => {
      result.current.subscribe(
        { aggregate_ids: [["catalog"]] },
        subscriptionHandler
      );
    });

    act(() => {
      instances[0].triggerOpen();
    });

    act(() => {
      instances[0].triggerMessage({
        type: "subscribed",
        sub_id: "0",
        items: [{ id: ["catalog"], data: { steps: {} }, sequence: 0 }],
      });
      instances[0].triggerMessage({
        type: "step_registered",
        sub_id: "0",
        data: { step: { id: "step-1" } },
      });
      instances[0].triggerMessage({ type: "step_registered", data: {} });
    });

    expect(subscriptionHandler).toHaveBeenCalledTimes(2);
    expect(fallbackHandler).toHaveBeenCalledTimes(1);
  });

  test("sends unsubscribe when removing an active subscription", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    let subscriptionId = "";
    act(() => {
      subscriptionId = result.current.subscribe({
        aggregate_ids: [["catalog"]],
      });
      instances[0].triggerOpen();
    });

    act(() => {
      result.current.unsubscribe(subscriptionId);
    });

    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "unsubscribe",
        data: { sub_id: "0" },
      })
    );
  });

  test("re-sends all subscriptions after reconnect", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      result.current.subscribe({ aggregate_ids: [["catalog"]] });
      result.current.subscribe({
        aggregate_ids: [["flow"]],
        event_types: ["flow_started"],
      });
    });

    act(() => {
      instances[0].triggerOpen();
    });

    act(() => {
      instances[0].triggerClose(1006);
    });

    act(() => {
      jest.advanceTimersByTime(WEBSOCKET.INITIAL_RECONNECT_DELAY);
    });

    expect(instances).toHaveLength(2);

    act(() => {
      instances[1].triggerOpen();
    });

    expect(instances[1].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: { aggregate_ids: [["catalog"]], sub_id: "0" },
      })
    );
    expect(instances[1].send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "subscribe",
        data: {
          aggregate_ids: [["flow"]],
          event_types: ["flow_started"],
          sub_id: "1",
        },
      })
    );
  });

  test("ignores pong messages", () => {
    const onEvent = jest.fn();
    renderHook(() => useWebSocketClient({ enabled: true, onEvent }));

    act(() => {
      instances[0].triggerOpen();
      instances[0].triggerMessage({ type: "pong" });
    });

    expect(onEvent).not.toHaveBeenCalled();
  });

  test("sends heartbeat pings while connected", () => {
    renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
      jest.advanceTimersByTime(WEBSOCKET.HEARTBEAT_INTERVAL);
    });

    expect(instances[0].send).toHaveBeenCalledWith(
      JSON.stringify({ type: "ping" })
    );
  });

  test("marks disconnected and does not reconnect on normal close", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
      instances[0].triggerClose(1000);
    });

    expect(result.current.connectionStatus).toBe("disconnected");
    act(() => {
      jest.advanceTimersByTime(WEBSOCKET.INITIAL_RECONNECT_DELAY);
    });
    expect(instances).toHaveLength(1);
  });

  test("reconnect() creates a new socket", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
      result.current.reconnect();
    });

    expect(instances).toHaveLength(2);
  });

  test("disconnects when disabled after being enabled", () => {
    const { result, rerender } = renderHook(
      ({ enabled }) => useWebSocketClient({ enabled }),
      { initialProps: { enabled: true } }
    );

    act(() => {
      instances[0].triggerOpen();
    });

    rerender({ enabled: false });

    expect(result.current.connectionStatus).toBe("disconnected");
  });

  test("does not connect when disabled", () => {
    renderHook(() => useWebSocketClient({ enabled: false }));
    expect(instances).toHaveLength(0);
  });

  test("marks disconnected on socket error", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
      instances[0].triggerError();
    });

    expect(result.current.connectionStatus).toBe("disconnected");
  });

  test("logs parse errors for invalid messages", () => {
    const onEvent = jest.fn();
    const consoleSpy = jest
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    renderHook(() => useWebSocketClient({ enabled: true, onEvent }));

    act(() => {
      instances[0].triggerOpen();
      instances[0].onmessage?.({ data: "{" });
    });

    expect(onEvent).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  test("stops reconnecting after max attempts", () => {
    const { result } = renderHook(() => useWebSocketClient({ enabled: true }));

    act(() => {
      instances[0].triggerOpen();
    });

    for (let i = 0; i < WEBSOCKET.MAX_RECONNECT_ATTEMPTS - 1; i += 1) {
      act(() => {
        instances[instances.length - 1].triggerClose(1006);
      });
      act(() => {
        jest.runOnlyPendingTimers();
      });
    }

    act(() => {
      instances[instances.length - 1].triggerClose(1006);
    });

    expect(result.current.connectionStatus).toBe("failed");
  });
});
