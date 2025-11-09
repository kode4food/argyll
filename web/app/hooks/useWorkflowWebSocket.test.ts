import { renderHook } from "@testing-library/react";
import { useWorkflowWebSocket } from "./useWorkflowWebSocket";
import { useWebSocketContext } from "./useWebSocketContext";
import { useWorkflowStore } from "../store/workflowStore";

jest.mock("./useWebSocketContext");
jest.mock("../store/workflowStore");

const mockUseWebSocketContext = useWebSocketContext as jest.MockedFunction<
  typeof useWebSocketContext
>;
const mockUseWorkflowStore = useWorkflowStore as jest.MockedFunction<
  typeof useWorkflowStore
>;

describe("useWorkflowWebSocket", () => {
  let mockSubscribe: jest.Mock;
  let mockRefreshExecutions: jest.Mock;
  let mockUpdateWorkflow: jest.Mock;
  let mockUpdateStepHealth: jest.Mock;
  let mockAddStep: jest.Mock;
  let mockRemoveStep: jest.Mock;

  beforeEach(() => {
    mockSubscribe = jest.fn();
    mockRefreshExecutions = jest.fn();
    mockUpdateWorkflow = jest.fn();
    mockUpdateStepHealth = jest.fn();
    mockAddStep = jest.fn();
    mockRemoveStep = jest.fn();

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

    mockUseWorkflowStore.mockImplementation((selector: any) => {
      const state = {
        selectedWorkflow: null,
        nextSequence: 0,
        workflowData: null,
        refreshExecutions: mockRefreshExecutions,
        updateWorkflowFromWebSocket: mockUpdateWorkflow,
        updateStepHealth: mockUpdateStepHealth,
        addStep: mockAddStep,
        removeStep: mockRemoveStep,
      };
      return selector(state);
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe("WebSocket subscription", () => {
    test("subscribes to engine events when no workflow selected", () => {
      renderHook(() => useWorkflowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({ engine_events: true });
    });

    test("subscribes to workflow events when workflow selected", () => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: "test-workflow",
          nextSequence: 42,
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });

      renderHook(() => useWorkflowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({
        engine_events: true,
        flow_id: "test-workflow",
        from_sequence: 42,
      });
    });

    test("uses sequence 0 when nextSequence is 0", () => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: "test-workflow",
          nextSequence: 0,
          workflowData: {},
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });

      renderHook(() => useWorkflowWebSocket());

      expect(mockSubscribe).toHaveBeenCalledWith({
        engine_events: true,
        flow_id: "test-workflow",
        from_sequence: 0,
      });
    });
  });

  describe("Engine events", () => {
    test("processes step_registered event regardless of selected workflow", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

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
            aggregate_id: [],
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
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_unregistered",
            data: { step_id: "test-step" },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: [],
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
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_health_changed",
            data: {
              step_id: "test-step",
              health_status: "healthy",
              health_error: undefined,
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: [],
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
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_health_changed",
            data: {
              step_id: "test-step",
              health_status: "unhealthy",
              health_error: "Connection failed",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: [],
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

  describe("Workflow events", () => {
    beforeEach(() => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: "test-workflow",
          nextSequence: 0,
          workflowData: {
            id: "test-workflow",
            status: "active",
            state: {},
          },
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });
    });

    test("processes workflow_started event", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      const startedAt = new Date().toISOString();
      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "workflow_started",
            data: {
              flow_id: "test-workflow",
              started_at: startedAt,
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).toHaveBeenCalledWith({
        status: "active",
        started_at: startedAt,
      });
    });

    test("processes attribute_set event with step provenance", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "attribute_set",
            data: {
              flow_id: "test-workflow",
              step_id: "producer-step",
              key: "result",
              value: "test-value",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).toHaveBeenCalledWith({
        state: {
          result: { value: "test-value", step: "producer-step" },
        },
      });
    });

    test("processes attribute_set without overwriting existing state", () => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: "test-workflow",
          nextSequence: 0,
          workflowData: {
            id: "test-workflow",
            status: "active",
            state: {
              existing: { value: "existing-value", step: "existing-step" },
            },
          },
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });

      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "attribute_set",
            data: {
              flow_id: "test-workflow",
              step_id: "new-step",
              key: "new_attr",
              value: "new-value",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).toHaveBeenCalledWith({
        state: {
          existing: { value: "existing-value", step: "existing-step" },
          new_attr: { value: "new-value", step: "new-step" },
        },
      });
    });

    test("processes workflow_completed event", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      const completedAt = new Date().toISOString();
      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "workflow_completed",
            data: {
              flow_id: "test-workflow",
              completed_at: completedAt,
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).toHaveBeenCalledWith({
        status: "completed",
        completed_at: completedAt,
      });
    });

    test("processes workflow_failed event", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      const failedAt = new Date().toISOString();
      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "workflow_failed",
            data: {
              flow_id: "test-workflow",
              error: "Test error",
              failed_at: failedAt,
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).toHaveBeenCalledWith({
        status: "failed",
        error_state: {
          message: "Test error",
          step_id: "",
          timestamp: failedAt,
        },
        completed_at: failedAt,
      });
    });

    test("refreshes executions when step_completed", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_completed",
            data: {
              flow_id: "test-workflow",
              step_id: "test-step",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockRefreshExecutions).toHaveBeenCalledWith("test-workflow");
    });

    test("refreshes executions for step_started, step_failed, step_skipped", () => {
      const eventTypes = ["step_started", "step_failed", "step_skipped"];

      eventTypes.forEach((eventType, index) => {
        mockRefreshExecutions.mockClear();

        const { rerender } = renderHook(() => useWorkflowWebSocket());

        mockUseWebSocketContext.mockReturnValue({
          events: [
            {
              type: eventType,
              data: {
                flow_id: "test-workflow",
                step_id: "test-step",
              },
              timestamp: Date.now(),
              sequence: index + 1,
              aggregate_id: ["test-workflow"],
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

        expect(mockRefreshExecutions).toHaveBeenCalledWith("test-workflow");
      });
    });
  });

  describe("Event filtering", () => {
    beforeEach(() => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: "test-workflow",
          nextSequence: 0,
          workflowData: {
            id: "test-workflow",
            status: "active",
            state: {},
          },
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });
    });

    test("ignores workflow events when no workflow selected", () => {
      mockUseWorkflowStore.mockImplementation((selector: any) => {
        const state = {
          selectedWorkflow: null,
          nextSequence: 0,
          workflowData: null,
          refreshExecutions: mockRefreshExecutions,
          updateWorkflowFromWebSocket: mockUpdateWorkflow,
          updateStepHealth: mockUpdateStepHealth,
          addStep: mockAddStep,
          removeStep: mockRemoveStep,
        };
        return selector(state);
      });

      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "workflow_started",
            data: {
              flow_id: "test-workflow",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["test-workflow"],
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

      expect(mockUpdateWorkflow).not.toHaveBeenCalled();
    });

    test("ignores events for different workflow", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "workflow_started",
            data: {
              flow_id: "other-workflow",
            },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: ["other-workflow"],
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

      expect(mockUpdateWorkflow).not.toHaveBeenCalled();
    });
  });

  describe("Event processing", () => {
    test("only processes new events", () => {
      const { rerender } = renderHook(() => useWorkflowWebSocket());

      mockUseWebSocketContext.mockReturnValue({
        events: [
          {
            type: "step_registered",
            data: { step: { id: "step-1" } },
            timestamp: Date.now(),
            sequence: 1,
            aggregate_id: [],
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
            aggregate_id: [],
          },
          {
            type: "step_registered",
            data: { step: { id: "step-2" } },
            timestamp: Date.now(),
            sequence: 2,
            aggregate_id: [],
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
      const { rerender } = renderHook(() => useWorkflowWebSocket());

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
      expect(mockUpdateWorkflow).not.toHaveBeenCalled();
    });
  });
});
