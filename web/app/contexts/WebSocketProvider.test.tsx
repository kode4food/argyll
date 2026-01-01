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
    loadFlows: jest.fn(),
    addStep: jest.fn(),
    updateStep: jest.fn(),
    removeStep: jest.fn(),
    updateStepHealth: jest.fn(),
    initializeExecutions: jest.fn(),
    updateExecution: jest.fn(),
    updateWorkItem: jest.fn(),
    updateFlowFromWebSocket: jest.fn(),
    setEngineSocketStatus: jest.fn(),
    engineReconnectRequest: 0,
  },
}));

jest.mock("@/app/hooks/useWebSocketClient");

describe("WebSocketProvider", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = flowStore.__storeState;
    (globalThis as any).__websocketStoreState.selectedFlow = "flow-1";
    (globalThis as any).__websocketStoreState.engineReconnectRequest = 0;
  });

  test("subscribes to engine and flow aggregates", () => {
    const engineSubscribe = jest.fn();
    const flowSubscribe = jest.fn();

    useWebSocketClientMock
      .mockImplementationOnce(() => ({
        connectionStatus: "connected",
        reconnectAttempt: 0,
        subscribe: engineSubscribe,
        reconnect: jest.fn(),
      }))
      .mockImplementationOnce(() => ({
        connectionStatus: "connected",
        reconnectAttempt: 0,
        subscribe: flowSubscribe,
        reconnect: jest.fn(),
      }));

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    expect(engineSubscribe).toHaveBeenCalledWith({
      aggregate_id: ["engine"],
      event_types: [
        "step_registered",
        "step_unregistered",
        "step_updated",
        "step_health_changed",
        "flow_activated",
        "flow_deactivated",
      ],
    });

    expect(flowSubscribe).toHaveBeenCalledWith({
      aggregate_id: ["flow", "flow-1"],
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
      ],
    });
  });

  test("disables flow socket when no flow is selected", () => {
    const engineSubscribe = jest.fn();
    const flowSubscribe = jest.fn();
    const flowStore = require("@/app/store/flowStore");
    (globalThis as any).__websocketStoreState = flowStore.__storeState;
    (globalThis as any).__websocketStoreState.selectedFlow = null;

    useWebSocketClientMock
      .mockImplementationOnce(() => ({
        connectionStatus: "connected",
        reconnectAttempt: 0,
        subscribe: engineSubscribe,
        reconnect: jest.fn(),
      }))
      .mockImplementationOnce(() => ({
        connectionStatus: "connected",
        reconnectAttempt: 0,
        subscribe: flowSubscribe,
        reconnect: jest.fn(),
      }));

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    expect(useWebSocketClientMock.mock.calls[1][0]?.enabled).toBe(false);
  });

  test("dispatches engine events to step handlers", () => {
    const engineClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };
    const flowClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };

    const clientOptions: Array<{ onEvent?: (event: any) => void }> = [];
    useWebSocketClientMock
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return engineClient;
      })
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return flowClient;
      });

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    clientOptions[0].onEvent?.({
      type: "step_registered",
      data: { step: { id: "step-1" } },
    });
    clientOptions[0].onEvent?.({
      type: "step_unregistered",
      data: { step_id: "step-2" },
    });
    clientOptions[0].onEvent?.({
      type: "step_health_changed",
      data: { step_id: "step-3", status: "healthy" },
    });
    clientOptions[0].onEvent?.({
      type: "flow_activated",
      data: { flow_id: "flow-1" },
    });
    clientOptions[0].onEvent?.({
      type: "flow_deactivated",
      data: { flow_id: "flow-1" },
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.addStep).toHaveBeenCalledWith({
      id: "step-1",
    });
    expect(flowStore.__storeState.removeStep).toHaveBeenCalledWith("step-2");
    expect(flowStore.__storeState.updateStepHealth).toHaveBeenCalledWith(
      "step-3",
      "healthy",
      undefined
    );
    expect(flowStore.__storeState.loadFlows).toHaveBeenCalledTimes(2);
  });

  test("dispatches flow events to execution and flow updates", () => {
    const engineClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };
    const flowClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };

    const clientOptions: Array<{ onEvent?: (event: any) => void }> = [];
    useWebSocketClientMock
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return engineClient;
      })
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return flowClient;
      });

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    clientOptions[1].onEvent?.({
      type: "flow_started",
      data: { flow_id: "flow-1", plan: { steps: { a: {} } } },
      timestamp: Date.now(),
    });
    clientOptions[1].onEvent?.({
      type: "step_started",
      data: { flow_id: "flow-1", step_id: "step-1", inputs: {} },
      timestamp: Date.now(),
    });
    clientOptions[1].onEvent?.({
      type: "step_completed",
      data: { flow_id: "flow-1", step_id: "step-1", outputs: {} },
      timestamp: Date.now(),
    });
    clientOptions[1].onEvent?.({
      type: "step_failed",
      data: { flow_id: "flow-1", step_id: "step-2", error: "boom" },
      timestamp: Date.now(),
    });
    clientOptions[1].onEvent?.({
      type: "step_skipped",
      data: { flow_id: "flow-1", step_id: "step-3" },
      timestamp: Date.now(),
    });
    clientOptions[1].onEvent?.({
      type: "attribute_set",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        key: "result",
        value: { ok: true },
      },
    });
    clientOptions[1].onEvent?.({
      type: "flow_completed",
      data: { flow_id: "flow-1" },
    });
    clientOptions[1].onEvent?.({
      type: "flow_failed",
      data: { flow_id: "flow-1", error: "bad" },
    });

    const flowStore = require("@/app/store/flowStore");
    expect(flowStore.__storeState.initializeExecutions).toHaveBeenCalledWith(
      "flow-1",
      { steps: { a: {} } }
    );
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
    expect(flowStore.__storeState.updateFlowFromWebSocket).toHaveBeenCalledWith(
      expect.objectContaining({
        state: expect.objectContaining({
          result: { value: { ok: true }, step: "step-1" },
        }),
      })
    );
    expect(flowStore.__storeState.updateFlowFromWebSocket).toHaveBeenCalledWith(
      expect.objectContaining({ status: "completed" })
    );
    expect(flowStore.__storeState.updateFlowFromWebSocket).toHaveBeenCalledWith(
      expect.objectContaining({ status: "failed" })
    );
  });

  test("dispatches work item events to updateWorkItem", () => {
    const engineClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };
    const flowClient = {
      connectionStatus: "connected",
      reconnectAttempt: 0,
      subscribe: jest.fn(),
      reconnect: jest.fn(),
    };

    const clientOptions: Array<{ onEvent?: (event: any) => void }> = [];
    useWebSocketClientMock
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return engineClient;
      })
      .mockImplementationOnce((options) => {
        clientOptions.push(options || {});
        return flowClient;
      });

    render(
      <WebSocketProvider>
        <div>child</div>
      </WebSocketProvider>
    );

    clientOptions[1].onEvent?.({
      type: "work_started",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-1",
        inputs: { key: "value" },
      },
    });
    clientOptions[1].onEvent?.({
      type: "work_succeeded",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-2",
        outputs: { result: "done" },
      },
    });
    clientOptions[1].onEvent?.({
      type: "work_failed",
      data: {
        flow_id: "flow-1",
        step_id: "step-1",
        token: "token-3",
        error: "something went wrong",
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
      { status: "completed", outputs: { result: "done" } }
    );
    expect(flowStore.__storeState.updateWorkItem).toHaveBeenCalledWith(
      "step-1",
      "token-3",
      { status: "failed", error: "something went wrong" }
    );
  });

  test("writes engine connection status to store", () => {
    useWebSocketClientMock
      .mockImplementationOnce(() => ({
        connectionStatus: "reconnecting",
        reconnectAttempt: 2,
        subscribe: jest.fn(),
        reconnect: jest.fn(),
      }))
      .mockImplementationOnce(() => ({
        connectionStatus: "connected",
        reconnectAttempt: 0,
        subscribe: jest.fn(),
        reconnect: jest.fn(),
      }));

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
