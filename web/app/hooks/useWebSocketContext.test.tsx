import React from "react";
import { act, render, waitFor } from "@testing-library/react";

import { WebSocketProvider, useWebSocketContext } from "./useWebSocketContext";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 3;

  url: string;
  readyState = MockWebSocket.CONNECTING;
  sent: string[] = [];
  onopen: ((ev?: any) => void) | null = null;
  onclose: ((ev: any) => void) | null = null;
  onerror: ((ev: any) => void) | null = null;
  onmessage: ((ev: any) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code: 1000 });
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.({});
  }

  fail(code = 1006) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.({ code });
  }

  message(payload: any) {
    const data =
      typeof payload === "string" ? payload : JSON.stringify(payload);
    this.onmessage?.({ data });
  }
}

const originalWebSocket = global.WebSocket;

function renderWithProvider() {
  let ctx: ReturnType<typeof useWebSocketContext> | null = null;

  const Consumer = () => {
    ctx = useWebSocketContext();
    return null;
  };

  render(
    <WebSocketProvider>
      <Consumer />
    </WebSocketProvider>
  );

  return () => {
    if (!ctx) {
      throw new Error("context not set");
    }
    return ctx;
  };
}

beforeAll(() => {
  // Provide defaults expected by the hook
  process.env.NEXT_PUBLIC_WS_URL = "ws://example.com";
  process.env.NEXT_PUBLIC_API_URL = "http://example.com";
  // @ts-expect-error mock
  global.WebSocket = MockWebSocket;
});

afterAll(() => {
  global.WebSocket = originalWebSocket;
});

afterEach(() => {
  MockWebSocket.instances = [];
});

it("throws when used outside provider", () => {
  const Outside = () => {
    useWebSocketContext();
    return null;
  };
  expect(() => render(<Outside />)).toThrow(
    "useWebSocketContext must be used within a WebSocketProvider"
  );
});

it("sends subscription when socket is already open", async () => {
  const getCtx = renderWithProvider();
  const socket = MockWebSocket.instances.at(-1);
  expect(socket).toBeDefined();

  act(() => socket?.open());
  const ctx = getCtx();

  act(() =>
    ctx.subscribe({
      engine_events: true,
      flow_id: "wf-1",
      event_types: ["flow_started"],
      from_sequence: 5,
    })
  );

  await waitFor(() => expect(socket?.sent).toHaveLength(1));
  const payload = JSON.parse(socket!.sent[0]);
  expect(payload.type).toBe("subscribe");
  expect(payload.data).toMatchObject({
    engine_events: true,
    flow_id: "wf-1",
    event_types: ["flow_started"],
    from_sequence: 5,
  });
});

it("trims events based on consumer cursor", async () => {
  const getCtx = renderWithProvider();
  const socket = MockWebSocket.instances.at(-1);
  expect(socket).toBeDefined();

  act(() => socket?.open());
  let ctx = getCtx();

  const consumerId = ctx.registerConsumer();
  act(() => ctx.updateConsumerCursor(consumerId, 1));

  act(() =>
    socket?.message({
      type: "event",
      data: { first: true },
      timestamp: 0,
      sequence: 0,
      id: ["evt-0"],
    })
  );

  act(() =>
    socket?.message({
      type: "event",
      data: { second: true },
      timestamp: 0,
      sequence: 1,
      id: ["evt-1"],
    })
  );

  await waitFor(() => {
    ctx = getCtx();
    expect(ctx.events.filter(Boolean).length).toBe(1);
  });

  const compacted = ctx.events.filter(Boolean);
  expect(compacted).toHaveLength(1);
  expect(compacted[0]).toMatchObject({
    id: ["evt-1"],
    data: { second: true },
  });
});

it("schedules reconnect after abnormal close", async () => {
  jest.useFakeTimers();
  const getCtx = renderWithProvider();
  const socket = MockWebSocket.instances.at(-1);
  expect(socket).toBeDefined();

  act(() => socket?.open());
  let ctx = getCtx();
  expect(ctx.connectionStatus).toBe("connected");

  act(() => socket?.fail(1006));

  await waitFor(() => {
    ctx = getCtx();
    expect(ctx.connectionStatus).toBe("reconnecting");
  });

  act(() => {
    jest.runOnlyPendingTimers();
  });
  jest.useRealTimers();
});
