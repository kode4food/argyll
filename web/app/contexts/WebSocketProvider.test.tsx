import { render } from "@testing-library/react";
import WebSocketProvider from "./WebSocketProvider";
import { useWebSocketClient } from "@/app/hooks/useWebSocketClient";

const useWebSocketClientMock = useWebSocketClient as jest.MockedFunction<
  typeof useWebSocketClient
>;

jest.mock("@/app/store/flowStore", () => ({
  useFlowStore: Object.assign(
    (selector: (state: any) => unknown) => {
      const state = (globalThis as any).__websocketStoreState;
      return selector(state);
    },
    {
      getState: () => (globalThis as any).__websocketStoreState,
    }
  ),
  __storeState: {
    selectedFlow: "flow-1",
    flowData: {
      id: "flow-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    },
    flows: [
      {
        id: "flow-1",
        status: "active",
        timestamp: "2024-01-01T00:00:00Z",
      },
    ],
    visibleFlowIDs: [],
    loadSteps: jest.fn(),
    addFlow: jest.fn(),
    addStep: jest.fn(),
    updateStep: jest.fn(),
    removeStep: jest.fn(),
    updateStepHealth: jest.fn(),
    initializeExecutions: jest.fn(),
    updateExecution: jest.fn(),
    updateWorkItem: jest.fn(),
    updateFlowData: jest.fn(),
    setFlowNotFound: jest.fn(),
    setEngineSocketStatus: jest.fn(),
    engineReconnectRequest: 0,
  },
}));

jest.mock("@/app/hooks/useWebSocketClient");

function makeClient(overrides?: Record<string, any>) {
  const subscribe = jest.fn((_subscription, _handler) => {
    return String(subscribe.mock.calls.length);
  });

  return {
    connectionStatus: "connected",
    reconnectAttempt: 0,
    subscribe,
    unsubscribe: jest.fn(),
    reconnect: jest.fn(),
    ...overrides,
  };
}

describe("WebSocketProvider", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = {
      ...flowStore.__storeState,
      flowData: {
        ...flowStore.__storeState.flowData,
      },
      flows: [...flowStore.__storeState.flows],
    };
    (globalThis as any).__websocketStoreState.selectedFlow = "flow-1";
    (globalThis as any).__websocketStoreState.engineReconnectRequest = 0;
  });

  test("subscribes to catalog, node, and selected flow", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    expect(useWebSocketClientMock).toHaveBeenCalledWith({
      enabled: true,
    });

    expect(client.subscribe).toHaveBeenNthCalledWith(
      1,
      {
        aggregate_ids: [["catalog"]],
        include_state: true,
        event_types: ["step_registered", "step_unregistered", "step_updated"],
      },
      expect.any(Function)
    );

    expect(client.subscribe).toHaveBeenNthCalledWith(
      2,
      {
        aggregate_ids: [["node"]],
        include_state: false,
        event_types: ["step_health_changed"],
      },
      expect.any(Function)
    );

    expect(client.subscribe).toHaveBeenNthCalledWith(
      3,
      {
        aggregate_ids: [["flow", "flow-1"]],
        include_state: true,
        event_types: [
          "flow_started",
          "step_started",
          "step_completed",
          "step_failed",
          "step_skipped",
          "attribute_set",
          "flow_completed",
          "flow_failed",
          "work_started",
          "work_succeeded",
          "work_failed",
          "work_not_completed",
          "retry_scheduled",
        ],
      },
      expect.any(Function)
    );
  });

  test("subscribes to flow summary when the list is visible", () => {
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = {
      ...flowStore.__storeState,
      visibleFlowIDs: ["flow-1"],
      flows: [...flowStore.__storeState.flows],
    };

    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    expect(client.subscribe).toHaveBeenNthCalledWith(
      3,
      {
        aggregate_ids: [["flow", "flow-1"]],
        include_state: false,
        event_types: ["flow_started", "flow_completed", "flow_failed"],
      },
      expect.any(Function)
    );
  });

  test("does not subscribe to selected flow when no flow is selected", () => {
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = {
      ...flowStore.__storeState,
      selectedFlow: null,
      flows: [...flowStore.__storeState.flows],
    };

    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    expect(client.subscribe).toHaveBeenCalledTimes(2);
  });

  test("dispatches catalog events to step handlers", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const catalogHandler = client.subscribe.mock.calls[0][1];

    catalogHandler({
      type: "step_registered",
      data: { step: { id: "step-1" } },
    });
    catalogHandler({
      type: "step_unregistered",
      data: { step_id: "step-2" },
    });
    catalogHandler({
      type: "step_updated",
      data: { step: { id: "step-3" } },
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.addStep).toHaveBeenCalledWith({
      id: "step-1",
    });
    expect(flowStore.__storeState.removeStep).toHaveBeenCalledWith("step-2");
    expect(flowStore.__storeState.updateStep).toHaveBeenCalledWith({
      id: "step-3",
    });
  });

  test("dispatches node events to health handlers", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const nodeHandler = client.subscribe.mock.calls[1][1];
    nodeHandler({
      type: "step_health_changed",
      data: { node_id: "node-1", step_id: "step-3", status: "healthy" },
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.updateStepHealth).toHaveBeenCalledWith(
      "node-1",
      "step-3",
      "healthy",
      undefined
    );
  });

  test("dispatches flow summary events to flow store", () => {
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = {
      ...flowStore.__storeState,
      visibleFlowIDs: ["flow-1"],
      flowData: {
        ...flowStore.__storeState.flowData,
      },
      flows: [...flowStore.__storeState.flows],
    };

    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const summaryHandler = client.subscribe.mock.calls[2][1];
    const startedAt = Date.parse("2024-01-02T00:00:00Z");
    const completedAt = Date.parse("2024-01-03T00:00:00Z");
    const failedAt = Date.parse("2024-01-04T00:00:00Z");

    summaryHandler({
      type: "flow_started",
      data: { flow_id: "flow-2" },
      timestamp: startedAt,
    });
    summaryHandler({
      type: "flow_completed",
      data: { flow_id: "flow-1" },
      timestamp: completedAt,
    });
    summaryHandler({
      type: "flow_failed",
      data: { flow_id: "flow-1", error: "bad" },
      timestamp: failedAt,
    });

    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(1, {
      id: "flow-2",
      status: "active",
      timestamp: "2024-01-02T00:00:00.000Z",
    });
    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(2, {
      id: "flow-1",
      status: "completed",
      timestamp: "2024-01-03T00:00:00.000Z",
    });
    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(3, {
      id: "flow-1",
      status: "failed",
      timestamp: "2024-01-04T00:00:00.000Z",
      error: "bad",
    });
  });

  test("dispatches flow events to execution and flow updates", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const flowHandler = client.subscribe.mock.calls[2][1];

    flowHandler({
      type: "flow_started",
      data: { flow_id: "flow-1", plan: { steps: { a: {} } } },
      timestamp: Date.parse("2024-01-05T00:00:00Z"),
    });
    flowHandler({
      type: "step_started",
      data: { flow_id: "flow-1", step_id: "step-1", inputs: {} },
      timestamp: Date.now(),
    });
    flowHandler({
      type: "step_completed",
      data: { flow_id: "flow-1", step_id: "step-1", outputs: {} },
      timestamp: Date.now(),
    });
    flowHandler({
      type: "step_failed",
      data: { flow_id: "flow-1", step_id: "step-2", error: "boom" },
      timestamp: Date.now(),
    });
    flowHandler({
      type: "step_skipped",
      data: { flow_id: "flow-1", step_id: "step-3" },
      timestamp: Date.now(),
    });
    flowHandler({
      type: "attribute_set",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        key: "result",
        value: { ok: true },
      },
    });
    flowHandler({
      type: "flow_completed",
      data: { flow_id: "flow-1" },
      timestamp: Date.parse("2024-01-06T00:00:00Z"),
    });
    flowHandler({
      type: "flow_failed",
      data: { flow_id: "flow-1", error: "bad" },
      timestamp: Date.parse("2024-01-07T00:00:00Z"),
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.initializeExecutions).toHaveBeenCalledWith(
      "flow-1",
      { steps: { a: {} } }
    );
    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(1, {
      id: "flow-1",
      status: "active",
      timestamp: "2024-01-05T00:00:00.000Z",
    });
    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(2, {
      id: "flow-1",
      status: "completed",
      timestamp: "2024-01-06T00:00:00.000Z",
    });
    expect(flowStore.__storeState.addFlow).toHaveBeenNthCalledWith(3, {
      id: "flow-1",
      status: "failed",
      timestamp: "2024-01-07T00:00:00.000Z",
      error: "bad",
    });
    expect(flowStore.__storeState.updateExecution).toHaveBeenCalledWith(
      "step-1",
      expect.objectContaining({ status: "active" })
    );
    expect(flowStore.__storeState.updateExecution).toHaveBeenCalledWith(
      "step-2",
      expect.objectContaining({ status: "failed" })
    );
    expect(flowStore.__storeState.updateExecution).toHaveBeenCalledWith(
      "step-3",
      expect.objectContaining({ status: "skipped" })
    );
    expect(flowStore.__storeState.updateFlowData).toHaveBeenCalledWith(
      expect.objectContaining({
        state: expect.objectContaining({
          result: { value: { ok: true }, step: "step-1" },
        }),
      })
    );
    expect(flowStore.__storeState.updateFlowData).toHaveBeenCalledWith(
      expect.objectContaining({
        status: "completed",
        completed_at: "2024-01-06T00:00:00.000Z",
      })
    );
    expect(flowStore.__storeState.updateFlowData).toHaveBeenCalledWith(
      expect.objectContaining({
        status: "failed",
        completed_at: "2024-01-07T00:00:00.000Z",
      })
    );
  });

  test("dispatches work item events to updateWorkItem", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const flowHandler = client.subscribe.mock.calls[2][1];
    flowHandler({
      type: "work_started",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-1",
        inputs: { key: "value" },
      },
    });
    flowHandler({
      type: "work_succeeded",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-2",
        outputs: { result: "done" },
      },
    });
    flowHandler({
      type: "work_failed",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-3",
        error: "something went wrong",
      },
    });
    flowHandler({
      type: "retry_scheduled",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-3",
        retry_count: 1,
        next_retry_at: "2025-01-01T00:00:00Z",
        error: "retry scheduled",
      },
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.updateWorkItem).toHaveBeenCalledWith(
      "step-1",
      "token-1",
      { status: "active", inputs: { key: "value" } }
    );
    expect(flowStore.__storeState.updateWorkItem).toHaveBeenCalledWith(
      "step-1",
      "token-2",
      { status: "succeeded", outputs: { result: "done" } }
    );
    expect(flowStore.__storeState.updateWorkItem).toHaveBeenCalledWith(
      "step-1",
      "token-3",
      { status: "failed", error: "something went wrong" }
    );
    expect(flowStore.__storeState.updateWorkItem).toHaveBeenCalledWith(
      "step-1",
      "token-3",
      {
        status: "pending",
        retry_count: 1,
        next_retry_at: "2025-01-01T00:00:00Z",
        error: "retry scheduled",
      }
    );
  });

  test("marks selected flow missing when subscribed state is empty", () => {
    const client = makeClient();
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const flowHandler = client.subscribe.mock.calls[2][1];
    flowHandler({
      type: "subscribed",
      sub_id: "2",
      items: [],
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.setFlowNotFound).toHaveBeenCalledWith(
      "flow-1"
    );
    expect(flowStore.__storeState.updateFlowData).not.toHaveBeenCalled();
  });

  test("writes socket connection status to store", () => {
    const client = makeClient({
      connectionStatus: "reconnecting",
      reconnectAttempt: 2,
    });
    useWebSocketClientMock.mockReturnValue(client);

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.setEngineSocketStatus).toHaveBeenCalledWith(
      "reconnecting",
      2
    );
  });
});
