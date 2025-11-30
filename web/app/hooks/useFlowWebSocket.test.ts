import { renderHook } from "@testing-library/react";
import { useFlowWebSocket } from "./useFlowWebSocket";
import { useWebSocketContext } from "./useWebSocketContext";
import { useFlowStore } from "../store/flowStore";

jest.mock("./useWebSocketContext");
jest.mock("../store/flowStore");

const mockUseWebSocketContext = useWebSocketContext as jest.MockedFunction<
  typeof useWebSocketContext
>;
const mockUseFlowStore = useFlowStore as jest.MockedFunction<
  typeof useFlowStore
>;

describe("useFlowWebSocket", () => {
  let mockSubscribe: jest.Mock;
  let mockRefreshExecutions: jest.Mock;
  let mockUpdateFlow: jest.Mock;
  let mockUpdateStepHealth: jest.Mock;
  let mockAddStep: jest.Mock;
  let mockRemoveStep: jest.Mock;
  let mockInitializeExecutions: jest.Mock;
  let mockUpdateExecution: jest.Mock;

  beforeEach(() => {
    mockSubscribe = jest.fn();
    mockRefreshExecutions = jest.fn();
    mockUpdateFlow = jest.fn();
    mockUpdateStepHealth = jest.fn();
    mockAddStep = jest.fn();
    mockRemoveStep = jest.fn();
    mockInitializeExecutions = jest.fn();
    mockUpdateExecution = jest.fn();

    mockUseWebSocketContext.mockReturnValue({
      events: [],
      subscribe: mockSubscribe,
      isConnected: true,
      connectionStatus: "connected",
      reconnectAttempt: 0,
      reconnect: jest.fn(),
      registerConsumer: jest.fn(() => "test-consumer-id"),
      unregisterConsumer: jest.fn(),
      updateConsumerCursor: jest.fn(),
    });

    mockUseFlowStore.mockImplementation((selector: any) => {
      const state = {
        selectedFlow: null,
        nextSequence: 0,
        flowData: null,
        refreshExecutions: mockRefreshExecutions,
        updateFlowFromWebSocket: mockUpdateFlow,
        updateStepHealth: mockUpdateStepHealth,
        addStep: mockAddStep,
        removeStep: mockRemoveStep,
        initializeExecutions: mockInitializeExecutions,
        updateExecution: mockUpdateExecution,
      };
      return selector(state);
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe("WebSocket subscription", () => {
    test("subscribes to engine events when no flow selected", () => {
      renderHook(() => useFlowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({ engine_events: true });
    });

    test("subscribes to flow events when flow selected", () => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 42,
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      renderHook(() => useFlowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({
        engine_events: true,
        flow_id: "test-flow",
        from_sequence: 42,
      });
    });

    test("uses sequence 0 when nextSequence is 0", () => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: {},
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });

      renderHook(() => useFlowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({
        engine_events: true,
        flow_id: "test-flow",
        from_sequence: 0,
      });
    });
  });

  describe("Engine events", () => {
    test("processes step_registered event regardless of selected flow", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const step = {
        id: "test-step",
        name: "Test Step",
        type: "sync" as const,
        required: {},
        optional: {},
        output: {},
        version: "1.0.0",
        http: { endpoint: "http://test", timeout: 5000 },
      };

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_registered",
            data: { step },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockAddStep).toHaveBeenCalledWith(step);
    });

    test("processes step_unregistered event", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_unregistered",
            data: { step_id: "test-step" },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockRemoveStep).toHaveBeenCalledWith("test-step");
    });

    test("processes step_health_changed event", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_health_changed",
            data: {
              step_id: "test-step",
              status: "healthy",
              error: undefined,
            },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateStepHealth).toHaveBeenCalledWith(
        "test-step",
        "healthy",
        undefined
      );
    });

    test("processes step_health_changed with error", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_health_changed",
            data: {
              step_id: "test-step",
              status: "unhealthy",
              error: "Connection failed",
            },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateStepHealth).toHaveBeenCalledWith(
        "test-step",
        "unhealthy",
        "Connection failed"
      );
    });
  });

  describe("Flow events", () => {
    beforeEach(() => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: {
            id: "test-flow",
            status: "active",
            state: {},
          },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });
    });

    test("processes flow_started event", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const startedAt = new Date().toISOString();
      const plan = {
        steps: {
          "step-1": { name: "Step 1" },
          "step-2": { name: "Step 2" },
        },
      };

      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: { id: "test-flow" },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "flow_started",
            data: {
              flow_id: "test-flow",
              started_at: startedAt,
              plan,
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).toHaveBeenCalledWith({
        status: "active",
        started_at: startedAt,
      });
      expect(mockInitializeExecutions).toHaveBeenCalledWith("test-flow", plan);
    });

    test("processes attribute_set event with step provenance", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "attribute_set",
            data: {
              flow_id: "test-flow",
              step_id: "producer-step",
              key: "result",
              value: "test-value",
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).toHaveBeenCalledWith({
        state: {
          result: { value: "test-value", step: "producer-step" },
        },
      });
    });

    test("processes attribute_set without overwriting existing state", () => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: {
            id: "test-flow",
            status: "active",
            state: {
              existing: { value: "existing-value", step: "existing-step" },
            },
          },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "attribute_set",
            data: {
              flow_id: "test-flow",
              step_id: "new-step",
              key: "new_attr",
              value: "new-value",
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).toHaveBeenCalledWith({
        state: {
          existing: { value: "existing-value", step: "existing-step" },
          new_attr: { value: "new-value", step: "new-step" },
        },
      });
    });

    test("processes flow_completed event", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const completedAt = new Date().toISOString();
      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "flow_completed",
            data: {
              flow_id: "test-flow",
              completed_at: completedAt,
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).toHaveBeenCalledWith({
        status: "completed",
        completed_at: completedAt,
      });
    });

    test("processes flow_failed event", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const failedAt = new Date().toISOString();
      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "flow_failed",
            data: {
              flow_id: "test-flow",
              error: "Test error",
              failed_at: failedAt,
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).toHaveBeenCalledWith({
        status: "failed",
        error_state: {
          message: "Test error",
          step_id: "",
          timestamp: failedAt,
        },
        completed_at: failedAt,
      });
    });

    test("calls updateExecution for step_completed", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const timestamp = Date.now();
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: { id: "test-flow" },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_completed",
            data: {
              flow_id: "test-flow",
              step_id: "test-step",
            },
            timestamp,
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateExecution).toHaveBeenCalledWith(
        "test-step",
        expect.objectContaining({
          status: "completed",
        })
      );
    });

    test("calls updateExecution for step_started", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const timestamp = Date.now();
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: { id: "test-flow" },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_started",
            data: { flow_id: "test-flow", step_id: "test-step" },
            timestamp,
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateExecution).toHaveBeenCalledWith(
        "test-step",
        expect.objectContaining({
          status: "active",
        })
      );
    });

    test("calls updateExecution for step_failed", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const timestamp = Date.now();
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: { id: "test-flow" },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_failed",
            data: {
              flow_id: "test-flow",
              step_id: "test-step",
              error: "test error",
            },
            timestamp,
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateExecution).toHaveBeenCalledWith(
        "test-step",
        expect.objectContaining({
          status: "failed",
          error_message: "test error",
        })
      );
    });

    test("calls updateExecution for step_skipped", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      const timestamp = Date.now();
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: { id: "test-flow" },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_skipped",
            data: { flow_id: "test-flow", step_id: "test-step" },
            timestamp,
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateExecution).toHaveBeenCalledWith(
        "test-step",
        expect.objectContaining({
          status: "skipped",
        })
      );
    });
  });

  describe("Event filtering", () => {
    beforeEach(() => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: "test-flow",
          nextSequence: 0,
          flowData: {
            id: "test-flow",
            status: "active",
            state: {},
          },
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });
    });

    test("ignores flow events when no flow selected", () => {
      mockUseFlowStore.mockImplementation((selector: any) => {
        const state = {
          selectedFlow: null,
          nextSequence: 0,
          flowData: null,
          refreshExecutions: mockRefreshExecutions,
          updateFlowFromWebSocket: mockUpdateFlow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
          initializeExecutions: mockInitializeExecutions,
          updateExecution: mockUpdateExecution,
        };
        return selector(state);
      });

      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "flow_started",
            data: {
              flow_id: "test-flow",
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["test-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).not.toHaveBeenCalled();
    });

    test("ignores events for different flow", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "flow_started",
            data: {
              flow_id: "other-flow",
            },
            timestamp: Date.now(),
            sequence: 1,
            id: ["other-flow"],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockUpdateFlow).not.toHaveBeenCalled();
    });
  });

  describe("Event processing", () => {
    test("only processes new events", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_registered",
            data: { step: { id: "step-1" } },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();
      expect(mockAddStep).toHaveBeenCalledTimes(1);

      rerender();
      expect(mockAddStep).toHaveBeenCalledTimes(1);

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_registered",
            data: { step: { id: "step-1" } },
            timestamp: Date.now(),
            sequence: 1,
            id: [],
          },
          {
            type: "step_registered",
            data: { step: { id: "step-2" } },
            timestamp: Date.now(),
            sequence: 2,
            id: [],
          },
        ],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();
      expect(mockAddStep).toHaveBeenCalledTimes(2);
      expect(mockAddStep).toHaveBeenLastCalledWith({ id: "step-2" });
    });

    test("handles empty events array", () => {
      const { rerender } = renderHook(() => useFlowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [],
        subscribe: mockSubscribe,
        isConnected: true,
        connectionStatus: "connected",
        reconnectAttempt: 0,
        reconnect: jest.fn(),
        registerConsumer: jest.fn(() => "test-consumer-id"),
        unregisterConsumer: jest.fn(),
        updateConsumerCursor: jest.fn(),
      });

      rerender();

      expect(mockAddStep).not.toHaveBeenCalled();
      expect(mockUpdateFlow).not.toHaveBeenCalled();
    });
  });
});
