import { renderHook } from "@testing-library/react";
import {
  useFlowStore,
  useSteps,
  useFlows,
  useSelectedFlow,
  useFlowData,
  useExecutions,
  useResolvedAttributes,
  useFlowLoading,
  useFlowError,
} from "./flowStore";
import type { Step, FlowContext, ExecutionResult, FlowSummary } from "../api";

jest.mock("../api", () => ({
  ...jest.requireActual("../api"),
  api: {
    getEngine: jest.fn(),
    listFlowsPage: jest.fn(),
  },
}));

import { api, AttributeRole, AttributeType } from "../api";

const mockApi = api as jest.Mocked<typeof api>;

const toFlowSummary = (flow: FlowContext): FlowSummary => ({
  id: flow.id,
  status: flow.status,
  timestamp: flow.completed_at || flow.started_at,
  error: flow.error_state?.message,
});

describe("flowStore", () => {
  beforeEach(() => {
    useFlowStore.setState({
      steps: [],
      stepHealth: {},
      flows: [],
      flowsCursor: null,
      flowsHasMore: false,
      flowsLoading: false,
      selectedFlow: null,
      flowData: null,
      executions: [],
      resolvedAttributes: [],
      loading: false,
      error: null,
      flowNotFound: false,
    });
    jest.clearAllMocks();
  });

  describe("Flow sorting", () => {
    test("loadFlows sorts active flows first, then by start time", async () => {
      const completedOld: FlowContext = {
        id: "wf-1",
        status: "completed",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
        completed_at: "2024-01-01T01:00:00Z",
      };

      const activeOld: FlowContext = {
        id: "wf-2",
        status: "active",
        state: {},
        started_at: "2024-01-02T00:00:00Z",
      };

      const activeNew: FlowContext = {
        id: "wf-3",
        status: "active",
        state: {},
        started_at: "2024-01-03T00:00:00Z",
      };

      const completedNew: FlowContext = {
        id: "wf-4",
        status: "completed",
        state: {},
        started_at: "2024-01-04T00:00:00Z",
        completed_at: "2024-01-04T01:00:00Z",
      };

      mockApi.listFlowsPage.mockResolvedValue({
        flows: [
          {
            id: completedOld.id,
            status: completedOld.status,
            timestamp: completedOld.completed_at!,
          },
          {
            id: activeOld.id,
            status: activeOld.status,
            timestamp: activeOld.started_at,
          },
          {
            id: completedNew.id,
            status: completedNew.status,
            timestamp: completedNew.completed_at!,
          },
          {
            id: activeNew.id,
            status: activeNew.status,
            timestamp: activeNew.started_at,
          },
        ],
        count: 4,
        total: 4,
        has_more: false,
      });

      await useFlowStore.getState().loadFlows();
      const state = useFlowStore.getState();

      expect(state.flows[0].id).toBe("wf-3");
      expect(state.flows[1].id).toBe("wf-2");
      expect(state.flows[2].id).toBe("wf-4");
      expect(state.flows[3].id).toBe("wf-1");
    });
  });

  describe("Step management", () => {
    const mockStep: Step = {
      id: "step-1",
      name: "Test Step",
      type: "sync",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
        result: { role: AttributeRole.Output, type: AttributeType.String },
      },
      http: {
        endpoint: "http://localhost:8080/test",
        timeout: 5000,
      },
    };

    test("loadSteps fetches and sorts steps alphabetically", async () => {
      mockApi.getEngine.mockResolvedValue({
        steps: {
          "step-1": { ...mockStep, name: "Zebra Step" },
          "step-2": { ...mockStep, id: "step-2", name: "Alpha Step" },
          "step-3": { ...mockStep, id: "step-3", name: "Beta Step" },
        },
        health: {
          "node-1": {
            "step-1": { status: "healthy" },
          },
          "node-2": {
            "step-1": { status: "unhealthy", error: "Connection timeout" },
          },
        },
      });

      await useFlowStore.getState().loadSteps();
      const state = useFlowStore.getState();

      expect(state.steps).toHaveLength(3);
      expect(state.steps[0].name).toBe("Alpha Step");
      expect(state.steps[2].name).toBe("Zebra Step");
      expect(state.stepHealth["step-1"]).toEqual({
        status: "unhealthy",
        error: "node node-2: Connection timeout",
        nodes: {
          "node-1": { status: "healthy", error: undefined },
          "node-2": {
            status: "unhealthy",
            error: "Connection timeout",
          },
        },
      });
    });

    test("loadSteps handles error", async () => {
      mockApi.getEngine.mockRejectedValue(new Error("Network error"));

      await useFlowStore.getState().loadSteps();
      const state = useFlowStore.getState();

      expect(state.error).toBe("Network error");
    });

    test("addStep creates new step", () => {
      useFlowStore.getState().addStep(mockStep);
      expect(useFlowStore.getState().steps).toHaveLength(1);
    });

    test("updateStep updates existing step", () => {
      useFlowStore.setState({ steps: [mockStep] });
      const updatedStep = { ...mockStep, name: "Updated Step" };
      useFlowStore.getState().updateStep(updatedStep);

      const state = useFlowStore.getState();
      expect(state.steps[0].name).toBe("Updated Step");
    });

    test("updateStep does nothing if step not found", () => {
      useFlowStore.setState({ steps: [mockStep] });
      const nonexistentStep = { ...mockStep, id: "step-999", name: "Unknown" };
      useFlowStore.getState().updateStep(nonexistentStep);

      const state = useFlowStore.getState();
      expect(state.steps).toHaveLength(1);
      expect(state.steps[0].name).toBe("Test Step");
    });

    test("upsertStep updates existing step", () => {
      useFlowStore.setState({ steps: [mockStep] });
      const updatedStep = { ...mockStep, name: "Updated Step" };

      useFlowStore.getState().upsertStep(updatedStep);
      const state = useFlowStore.getState();

      expect(state.steps).toHaveLength(1);
      expect(state.steps[0].name).toBe("Updated Step");
    });

    test("upsertStep adds missing step", () => {
      useFlowStore.getState().upsertStep(mockStep);
      const state = useFlowStore.getState();

      expect(state.steps).toHaveLength(1);
      expect(state.steps[0].id).toBe("step-1");
    });

    test("removeStep deletes step", () => {
      useFlowStore.setState({ steps: [mockStep] });
      useFlowStore.getState().removeStep("step-1");
      expect(useFlowStore.getState().steps).toHaveLength(0);
    });
  });

  describe("Flow management", () => {
    const mockFlow: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    test("loadFlows fetches flows", async () => {
      mockApi.listFlowsPage.mockResolvedValue({
        flows: [
          {
            id: mockFlow.id,
            status: mockFlow.status,
            timestamp: mockFlow.started_at,
          },
        ],
        count: 1,
        total: 1,
        has_more: false,
      });
      await useFlowStore.getState().loadFlows();
      expect(useFlowStore.getState().flows).toHaveLength(1);
    });

    test("loadFlows handles error", async () => {
      mockApi.listFlowsPage.mockRejectedValue(new Error("Network error"));

      await useFlowStore.getState().loadFlows();
      const state = useFlowStore.getState();

      expect(state.error).toBe("Network error");
    });

    test("addFlow adds flow", () => {
      useFlowStore.getState().addFlow(toFlowSummary(mockFlow));
      expect(useFlowStore.getState().flows).toHaveLength(1);
    });

    test("removeFlow deletes flow", () => {
      useFlowStore.setState({ flows: [toFlowSummary(mockFlow)] });
      useFlowStore.getState().removeFlow("wf-1");
      expect(useFlowStore.getState().flows).toHaveLength(0);
    });

    test("selectFlow sets selected flow", () => {
      useFlowStore.getState().selectFlow("wf-1");
      const state = useFlowStore.getState();

      expect(state.selectedFlow).toBe("wf-1");
      expect(state.flowData).toBeNull();
    });

    test("selectFlow with null clears selection", () => {
      useFlowStore.setState({ selectedFlow: "wf-1" });

      useFlowStore.getState().selectFlow(null);
      const state = useFlowStore.getState();

      expect(state.selectedFlow).toBeNull();
    });

    test("setFlowNotFound marks the selected flow missing", () => {
      useFlowStore.setState({
        selectedFlow: "wf-1",
        flowData: {
          id: "wf-1",
          status: "active",
          state: {},
          started_at: "2024-01-01T00:00:00Z",
        },
        executions: [
          {
            flow_id: "wf-1",
            step_id: "step-1",
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
          },
        ],
        resolvedAttributes: ["foo"],
        loading: true,
      });

      useFlowStore.getState().setFlowNotFound("wf-1");

      const state = useFlowStore.getState();
      expect(state.flowData).toBeNull();
      expect(state.executions).toEqual([]);
      expect(state.resolvedAttributes).toEqual([]);
      expect(state.loading).toBe(false);
      expect(state.flowNotFound).toBe(true);
    });

    test("selectFlow skips if same flow already selected with data", () => {
      useFlowStore.setState({
        selectedFlow: "wf-1",
        flowData: mockFlow,
        loading: false,
      });

      useFlowStore.getState().selectFlow("wf-1");
      const state = useFlowStore.getState();

      expect(state.loading).toBe(false);
    });
  });

  describe("Execution management", () => {
    test("initializeExecutions creates executions from plan", () => {
      const plan = {
        steps: {
          "step-1": {},
          "step-2": {},
        },
      };

      useFlowStore.getState().initializeExecutions("wf-1", plan);
      const state = useFlowStore.getState();

      expect(state.executions).toHaveLength(2);
      expect(state.executions[0].status).toBe("pending");
    });

    test("initializeExecutions handles empty plan", () => {
      useFlowStore.getState().initializeExecutions("wf-1", null);
      const state = useFlowStore.getState();

      expect(state.executions).toHaveLength(0);
    });

    test("updateExecution updates execution status", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "pending",
            inputs: {},
            started_at: "",
          },
        ],
        resolvedAttributes: [],
      });

      useFlowStore.getState().updateExecution("step-1", {
        status: "completed",
        outputs: { result: "value" },
      });

      const state = useFlowStore.getState();
      expect(state.executions[0].status).toBe("completed");
      expect(state.resolvedAttributes).toContain("result");
    });

    test("updateExecution does nothing if step not found", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "pending",
            inputs: {},
            started_at: "",
          },
        ],
        resolvedAttributes: [],
      });

      useFlowStore.getState().updateExecution("step-999", {
        status: "completed",
      });

      const state = useFlowStore.getState();
      expect(state.executions[0].status).toBe("pending");
    });

    test("updateWorkItem creates work item if not exists", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
          },
        ],
      });

      useFlowStore.getState().updateWorkItem("step-1", "token-1", {
        status: "active",
        inputs: { key: "value" },
      });

      const state = useFlowStore.getState();
      expect(state.executions[0].work_items).toBeDefined();
      expect(state.executions[0].work_items?.["token-1"]).toEqual({
        token: "token-1",
        status: "active",
        inputs: { key: "value" },
        retry_count: 0,
      });
    });

    test("updateWorkItem updates existing work item", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
            work_items: {
              "token-1": {
                token: "token-1",
                status: "active",
                inputs: { key: "value" },
                retry_count: 0,
              },
            },
          },
        ],
      });

      useFlowStore.getState().updateWorkItem("step-1", "token-1", {
        status: "completed",
        outputs: { result: "done" },
      });

      const state = useFlowStore.getState();
      expect(state.executions[0].work_items?.["token-1"]).toEqual({
        token: "token-1",
        status: "completed",
        inputs: { key: "value" },
        outputs: { result: "done" },
        retry_count: 0,
      });
    });

    test("updateWorkItem does nothing if step not found", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
          },
        ],
      });

      useFlowStore.getState().updateWorkItem("step-999", "token-1", {
        status: "completed",
      });

      const state = useFlowStore.getState();
      expect(state.executions[0].work_items).toBeUndefined();
    });

    test("updateWorkItem handles retry_token by removing old token and updating with new", () => {
      useFlowStore.setState({
        executions: [
          {
            step_id: "step-1",
            flow_id: "wf-1",
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
            work_items: {
              "token-1": {
                token: "token-1",
                status: "active",
                inputs: { key: "value" },
                retry_count: 0,
              },
            },
          },
        ],
      });

      // Retry with new token
      useFlowStore.getState().updateWorkItem(
        "step-1",
        "token-1",
        {
          status: "active",
          inputs: { key: "value" },
          retry_count: 1,
        },
        "token-2"
      );

      const state = useFlowStore.getState();
      // Old token should be removed
      expect(state.executions[0].work_items?.["token-1"]).toBeUndefined();
      // New token should exist with updated data
      expect(state.executions[0].work_items?.["token-2"]).toEqual({
        token: "token-2",
        status: "active",
        inputs: { key: "value" },
        retry_count: 1,
      });
      // Total work items should still be 1
      expect(Object.keys(state.executions[0].work_items || {}).length).toBe(1);
    });
  });

  describe("Step health updates", () => {
    test("updateStepHealth updates step health", () => {
      useFlowStore.setState({ stepHealth: {} });

      useFlowStore.getState().updateStepHealth("node-1", "step-1", "healthy");
      const state = useFlowStore.getState();

      expect(state.stepHealth["step-1"]).toEqual({
        status: "healthy",
        nodes: {
          "node-1": { status: "healthy" },
        },
      });
    });

    test("updateStepHealth updates with error", () => {
      useFlowStore
        .getState()
        .updateStepHealth("node-1", "step-1", "unhealthy", "Connection failed");
      const state = useFlowStore.getState();

      expect(state.stepHealth["step-1"]).toEqual({
        status: "unhealthy",
        error: "node node-1: Connection failed",
        nodes: {
          "node-1": {
            status: "unhealthy",
            error: "Connection failed",
          },
        },
      });
    });
  });

  describe("Flow updates", () => {
    const mockFlow: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    test("updateFlowData merges detail state", () => {
      useFlowStore.setState({
        flowData: mockFlow,
        flows: [toFlowSummary(mockFlow)],
        resolvedAttributes: [],
      });

      const update: Partial<FlowContext> = {
        status: "completed",
        state: { result: { value: "final", step: "final-step" } },
      };

      useFlowStore.getState().updateFlowData(update);

      const state = useFlowStore.getState();

      expect(state.flowData?.status).toBe("completed");
      expect(state.flowData?.state).toEqual({
        result: { value: "final", step: "final-step" },
      });
      expect(state.resolvedAttributes).toContain("result");
    });

    test("updateFlowData leaves flow summaries alone", () => {
      useFlowStore.setState({
        flowData: mockFlow,
        flows: [toFlowSummary(mockFlow)],
        resolvedAttributes: [],
      });

      const update: Partial<FlowContext> = {
        status: "completed",
      };

      useFlowStore.getState().updateFlowData(update);

      const state = useFlowStore.getState();
      expect(state.flows).toEqual([toFlowSummary(mockFlow)]);
    });

    test("addFlow replaces an existing summary", () => {
      useFlowStore.setState({
        flows: [toFlowSummary(mockFlow)],
      });

      useFlowStore.getState().addFlow({
        id: "wf-1",
        status: "completed",
        timestamp: "2024-01-02T00:00:00Z",
      });
      const state = useFlowStore.getState();

      expect(state.flows[0].status).toBe("completed");
      expect(state.flows[0].timestamp).toBe("2024-01-02T00:00:00Z");
    });
  });

  describe("WebSocket state handling", () => {
    test("setCatalogState updates steps from WebSocket", () => {
      const mockStep: Step = {
        id: "step-1",
        name: "Test Step",
        type: "sync",
        attributes: {},
        http: { endpoint: "http://localhost:8080/test", timeout: 5000 },
      };

      useFlowStore.getState().setCatalogState({ "step-1": mockStep });

      const state = useFlowStore.getState();
      expect(state.steps).toHaveLength(1);
      expect(state.steps[0].id).toBe("step-1");
    });

    test("setHealthState updates health from WebSocket", () => {
      useFlowStore.getState().setHealthState({
        "node-1": {
          "step-1": { status: "healthy" },
        },
        "node-2": {
          "step-1": { status: "unhealthy", error: "Connection failed" },
        },
      });

      const state = useFlowStore.getState();
      expect(state.stepHealth["step-1"]).toEqual({
        status: "unhealthy",
        error: "node node-2: Connection failed",
        nodes: {
          "node-1": { status: "healthy", error: undefined },
          "node-2": {
            status: "unhealthy",
            error: "Connection failed",
          },
        },
      });
    });

    test("setCatalogState handles empty state", () => {
      useFlowStore.getState().setCatalogState({});

      const state = useFlowStore.getState();
      expect(state.steps).toHaveLength(0);
    });

    test("setFlowState sets flow data from WebSocket", () => {
      useFlowStore.setState({ selectedFlow: "wf-1" });

      useFlowStore.getState().setFlowState({
        id: "wf-1",
        status: "active",
        attributes: { result: "value" },
        plan: { steps: { "step-1": {} } },
        executions: {
          "step-1": {
            status: "completed",
            inputs: { arg: "val" },
            outputs: { out: "result" },
          },
        },
        created_at: "2024-01-01T00:00:00Z",
      });

      const state = useFlowStore.getState();
      expect(state.flowData?.id).toBe("wf-1");
      expect(state.flowData?.status).toBe("active");
      expect(state.flowData?.state).toEqual({ result: "value" });
      expect(state.executions).toHaveLength(1);
      expect(state.executions[0].status).toBe("completed");
      expect(state.resolvedAttributes).toContain("result");
      expect(state.resolvedAttributes).toContain("out");
      expect(state.loading).toBe(false);
    });

    test("setFlowState updates the flow list entry", () => {
      useFlowStore.setState({
        selectedFlow: "wf-1",
        flows: [
          {
            id: "wf-1",
            status: "pending",
            timestamp: "2024-01-01T00:00:00Z",
          },
        ],
      });

      useFlowStore.getState().setFlowState({
        id: "wf-1",
        status: "completed",
        created_at: "2024-01-01T00:00:00Z",
        completed_at: "2024-01-02T00:00:00Z",
      });

      const state = useFlowStore.getState();
      expect(state.flows).toEqual([
        {
          id: "wf-1",
          status: "completed",
          timestamp: "2024-01-02T00:00:00Z",
          error: undefined,
        },
      ]);
    });

    test("setFlowState ignores unselected flow", () => {
      useFlowStore.setState({ selectedFlow: "wf-2" });

      useFlowStore.getState().setFlowState({
        id: "wf-1",
        status: "active",
      });

      const state = useFlowStore.getState();
      expect(state.flowData).toBeNull();
    });

    test("setFlowState handles error state", () => {
      useFlowStore.setState({ selectedFlow: "wf-1" });

      useFlowStore.getState().setFlowState({
        id: "wf-1",
        status: "failed",
        error: "Something went wrong",
      });

      const state = useFlowStore.getState();
      expect(state.flowData?.status).toBe("failed");
      expect(state.flowData?.error_state?.message).toBe("Something went wrong");
    });

    test("setEngineSocketStatus updates connection status", () => {
      useFlowStore.getState().setEngineSocketStatus("connected", 0);

      const state = useFlowStore.getState();
      expect(state.engineConnectionStatus).toBe("connected");
      expect(state.engineReconnectAttempt).toBe(0);
    });

    test("setEngineSocketStatus tracks reconnect attempts", () => {
      useFlowStore.getState().setEngineSocketStatus("connecting", 3);

      const state = useFlowStore.getState();
      expect(state.engineConnectionStatus).toBe("connecting");
      expect(state.engineReconnectAttempt).toBe(3);
    });

    test("requestEngineReconnect increments request counter", () => {
      const initialRequest = useFlowStore.getState().engineReconnectRequest;

      useFlowStore.getState().requestEngineReconnect();

      const state = useFlowStore.getState();
      expect(state.engineReconnectRequest).toBe(initialRequest + 1);
    });
  });

  describe("Selector hooks", () => {
    test("useSteps selector works", () => {
      const mockStep: Step = {
        id: "step-1",
        name: "Test",
        type: "sync",
        attributes: {},

        http: {
          endpoint: "http://localhost:8080/test",
          timeout: 5000,
        },
      };

      useFlowStore.setState({ steps: [mockStep] });
      const { result } = renderHook(() => useSteps());
      expect(result.current).toEqual([mockStep]);
    });

    test("useFlows selector works", () => {
      const mockFlow: FlowSummary = {
        id: "wf-1",
        status: "active",
        timestamp: "2024-01-01T00:00:00Z",
      };

      useFlowStore.setState({ flows: [mockFlow] });
      const { result } = renderHook(() => useFlows());
      expect(result.current).toEqual([mockFlow]);
    });

    test("useSelectedFlow selector works", () => {
      useFlowStore.setState({ selectedFlow: "wf-1" });
      const { result } = renderHook(() => useSelectedFlow());
      expect(result.current).toBe("wf-1");
    });

    test("useFlowData selector works", () => {
      const mockFlow: FlowContext = {
        id: "wf-1",
        status: "active",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
      };
      useFlowStore.setState({ flowData: mockFlow });
      const { result } = renderHook(() => useFlowData());
      expect(result.current).toEqual(mockFlow);
    });

    test("useExecutions selector works", () => {
      const executions: ExecutionResult[] = [
        {
          step_id: "step-1",
          flow_id: "wf-1",
          status: "completed",
          inputs: {},
          started_at: "2024-01-01T00:00:00Z",
        },
      ];
      useFlowStore.setState({ executions });
      const { result } = renderHook(() => useExecutions());
      expect(result.current).toEqual(executions);
    });

    test("useResolvedAttributes selector works", () => {
      const attrs = ["attr1", "attr2"];
      useFlowStore.setState({ resolvedAttributes: attrs });
      const { result } = renderHook(() => useResolvedAttributes());
      expect(result.current).toEqual(attrs);
    });

    test("useFlowLoading selector works", () => {
      useFlowStore.setState({ loading: true });
      const { result } = renderHook(() => useFlowLoading());
      expect(result.current).toBe(true);
    });

    test("useFlowError selector works", () => {
      useFlowStore.setState({ error: "Test error" });
      const { result } = renderHook(() => useFlowError());
      expect(result.current).toBe("Test error");
    });
  });
});
