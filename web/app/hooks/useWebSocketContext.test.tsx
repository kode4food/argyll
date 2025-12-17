import { act, renderHook } from "@testing-library/react";
import React from "react";
import { WebSocketProvider, useWebSocketContext } from "./useWebSocketContext";

class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
  url: string;
  sent: string[] = [];
  onopen: ((event?: any) => void) | null = null;
  onclose: ((event: any) => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: any) => void) | null = null;

  constructor(url: string) {
    this.url = url;
  }

  send(data: string) {
    this.sent.push(data);
  }

  close(event: any = { code: 1000 }) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(event);
  }
}

describe("useWebSocketContext", () => {
  const sockets: MockWebSocket[] = [];

  beforeAll(() => {
    const WebSocketMock: any = jest.fn((url: string) => {
      const socket = new MockWebSocket(url);
      sockets.push(socket);
      return socket;
    });
    WebSocketMock.CONNECTING = MockWebSocket.CONNECTING;
    WebSocketMock.OPEN = MockWebSocket.OPEN;
    WebSocketMock.CLOSING = MockWebSocket.CLOSING;
    WebSocketMock.CLOSED = MockWebSocket.CLOSED;

    global.WebSocket = WebSocketMock;
  });

  beforeEach(() => {
    jest.useFakeTimers();
    sockets.length = 0;
    jest.clearAllMocks();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  const wrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <WebSocketProvider>{children}</WebSocketProvider>
  );

  it("sends subscription on open", () => {
    const { result, unmount } = renderHook(() => useWebSocketContext(), {
      wrapper,
    });

    const socket = sockets[0];

    act(() => {
      socket.readyState = MockWebSocket.OPEN;
      socket.onopen?.();
    });

    act(() => {
      result.current.subscribe({ event_types: ["flow_started"] });
    });

    expect(socket.sent).toHaveLength(1);
    const payload = JSON.parse(socket.sent[0]);
    expect(payload.type).toBe("subscribe");
    expect(payload.data.event_types).toEqual(["flow_started"]);
    expect(result.current.isConnected).toBe(true);

    unmount();
  });

  it("prunes buffered events by cursor", () => {
    const { result, unmount } = renderHook(() => useWebSocketContext(), {
      wrapper,
    });
    const socket = sockets[0];

    act(() => {
      socket.readyState = MockWebSocket.OPEN;
      socket.onopen?.();
    });

    act(() => {
      socket.onmessage?.({
        data: JSON.stringify({
          type: "flow_started",
          data: {},
          timestamp: 1,
          sequence: 1,
          id: ["flow", "one"],
        }),
      });
    });

    expect(result.current.events).toHaveLength(1);

    const consumer = result.current.registerConsumer();
    act(() => {
      result.current.updateConsumerCursor(consumer, 1);
    });

    act(() => {
      socket.onmessage?.({
        data: JSON.stringify({
          type: "flow_completed",
          data: {},
          timestamp: 2,
          sequence: 2,
          id: ["flow", "one"],
        }),
      });
    });

    expect(result.current.events.filter(Boolean)).toHaveLength(1);
    expect(result.current.events.filter(Boolean)[0]?.type).toBe(
      "flow_completed"
    );

    unmount();
  });

  it("reconnects after unexpected close", () => {
    const { result, unmount } = renderHook(() => useWebSocketContext(), {
      wrapper,
    });
    const socket = sockets[0];

    act(() => {
      socket.readyState = MockWebSocket.OPEN;
      socket.onopen?.();
    });

    act(() => {
      socket.onclose?.({ code: 4000 });
    });

    expect(result.current.connectionStatus).toBe("reconnecting");

    act(() => {
      jest.runOnlyPendingTimers();
    });

    expect(sockets.length).toBe(2);
    expect(result.current.connectionStatus).toBe("connecting");

    unmount();
  });
});
