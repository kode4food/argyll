import { renderHook } from "@testing-library/react";
import {
  useWorkflowStore,
  useSteps,
  useWorkflows,
  useSelectedWorkflow,
  useWorkflowData,
  useExecutions,
  useResolvedAttributes,
  useWorkflowLoading,
  useWorkflowError,
  useIsWorkflowMode,
} from "./workflowStore";
import type { Step, WorkflowContext } from "../api";

jest.mock("../api", () => ({
  ...jest.requireActual("../api"),
  api: {
    getEngineState: jest.fn(),
    listWorkflows: jest.fn(),
    getWorkflowWithEvents: jest.fn(),
    getExecutions: jest.fn(),
  },
}));

import { api, AttributeRole, AttributeType } from "../api";

const mockApi = api as jest.Mocked<typeof api>;

describe("workflowStore", () => {
  beforeEach(() => {
    useWorkflowStore.setState({
      steps: [],
      stepHealth: {},
      workflows: [],
      selectedWorkflow: null,
      workflowData: null,
      executions: [],
      resolvedAttributes: [],
      loading: false,
      error: null,
      workflowNotFound: false,
      isWorkflowMode: false,
    });
    jest.clearAllMocks();
  });

  describe("Workflow sorting", () => {
    test("loadWorkflows sorts active workflows first, then by start time", async () => {
      const completedOld: WorkflowContext = {
        id: "wf-1",
        status: "completed",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
        completed_at: "2024-01-01T01:00:00Z",
      };

      const activeOld: WorkflowContext = {
        id: "wf-2",
        status: "active",
        state: {},
        started_at: "2024-01-02T00:00:00Z",
      };

      const activeNew: WorkflowContext = {
        id: "wf-3",
        status: "active",
        state: {},
        started_at: "2024-01-03T00:00:00Z",
      };

      const completedNew: WorkflowContext = {
        id: "wf-4",
        status: "completed",
        state: {},
        started_at: "2024-01-04T00:00:00Z",
        completed_at: "2024-01-04T01:00:00Z",
      };

      mockApi.listWorkflows.mockResolvedValue([
        completedOld,
        activeOld,
        completedNew,
        activeNew,
      ]);

      await useWorkflowStore.getState().loadWorkflows();
      const state = useWorkflowStore.getState();

      expect(state.workflows[0].id).toBe("wf-3");
      expect(state.workflows[1].id).toBe("wf-2");
      expect(state.workflows[2].id).toBe("wf-4");
      expect(state.workflows[3].id).toBe("wf-1");
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
      version: "1.0.0",
      http: {
        endpoint: "http://localhost:8080/test",
        timeout: 5000,
      },
    };

    test("loadSteps fetches and sorts steps alphabetically", async () => {
      mockApi.getEngineState.mockResolvedValue({
        steps: {
          "step-1": { ...mockStep, name: "Zebra Step" },
          "step-2": { ...mockStep, id: "step-2", name: "Alpha Step" },
          "step-3": { ...mockStep, id: "step-3", name: "Beta Step" },
        },
        health: {
          "step-1": { status: "healthy" },
          "step-2": { status: "unhealthy", error: "Connection timeout" },
        },
      });

      await useWorkflowStore.getState().loadSteps();
      const state = useWorkflowStore.getState();

      expect(state.steps).toHaveLength(3);
      expect(state.steps[0].name).toBe("Alpha Step");
      expect(state.steps[2].name).toBe("Zebra Step");
      expect(state.stepHealth["step-1"]).toEqual({ status: "healthy" });
    });

    test("loadSteps handles error", async () => {
      mockApi.getEngineState.mockRejectedValue(new Error("Network error"));

      await useWorkflowStore.getState().loadSteps();
      const state = useWorkflowStore.getState();

      expect(state.error).toBe("Network error");
    });

    test("addStep creates new step", () => {
      useWorkflowStore.getState().addStep(mockStep);
      expect(useWorkflowStore.getState().steps).toHaveLength(1);
    });

    test("addStep updates existing step", () => {
      useWorkflowStore.setState({ steps: [mockStep] });

      const updatedStep = { ...mockStep, version: "2.0.0" };
      useWorkflowStore.getState().addStep(updatedStep);

      const state = useWorkflowStore.getState();
      expect(state.steps).toHaveLength(1);
      expect(state.steps[0].version).toBe("2.0.0");
    });

    test("removeStep deletes step", () => {
      useWorkflowStore.setState({ steps: [mockStep] });
      useWorkflowStore.getState().removeStep("step-1");
      expect(useWorkflowStore.getState().steps).toHaveLength(0);
    });
  });

  describe("Workflow management", () => {
    const mockWorkflow: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    test("loadWorkflows fetches workflows", async () => {
      mockApi.listWorkflows.mockResolvedValue([mockWorkflow]);
      await useWorkflowStore.getState().loadWorkflows();
      expect(useWorkflowStore.getState().workflows).toHaveLength(1);
    });

    test("loadWorkflows handles error", async () => {
      mockApi.listWorkflows.mockRejectedValue(new Error("Network error"));

      await useWorkflowStore.getState().loadWorkflows();
      const state = useWorkflowStore.getState();

      expect(state.error).toBe("Network error");
    });

    test("addWorkflow adds workflow", () => {
      useWorkflowStore.getState().addWorkflow(mockWorkflow);
      expect(useWorkflowStore.getState().workflows).toHaveLength(1);
    });

    test("removeWorkflow deletes workflow", () => {
      useWorkflowStore.setState({ workflows: [mockWorkflow] });
      useWorkflowStore.getState().removeWorkflow("wf-1");
      expect(useWorkflowStore.getState().workflows).toHaveLength(0);
    });

    test("selectWorkflow sets selected workflow and mode", () => {
      useWorkflowStore.getState().selectWorkflow("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.selectedWorkflow).toBe("wf-1");
      expect(state.isWorkflowMode).toBe(true);
      expect(state.workflowData).toBeNull();
    });

    test("selectWorkflow with null clears selection", () => {
      useWorkflowStore.setState({
        selectedWorkflow: "wf-1",
        isWorkflowMode: true,
      });

      useWorkflowStore.getState().selectWorkflow(null);
      const state = useWorkflowStore.getState();

      expect(state.selectedWorkflow).toBeNull();
      expect(state.isWorkflowMode).toBe(false);
    });
  });

  describe("Workflow data loading", () => {
    const mockWorkflow: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: { attr1: "value1" },
      started_at: "2024-01-01T00:00:00Z",
      plan: {
        steps: {},
        attributes: {},
        goals: [],
        required: [],
      },
    };

    test("loadWorkflowData fetches workflow with executions", async () => {
      const mockExecutions = [
        {
          step_id: "step-1",
          status: "completed",
          outputs: { result: "value" },
        },
      ];

      mockApi.getWorkflowWithEvents.mockResolvedValue({
        workflow: mockWorkflow,
        executions: mockExecutions,
      });

      await useWorkflowStore.getState().loadWorkflowData("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.workflowData).toEqual(mockWorkflow);
      expect(state.executions).toEqual(mockExecutions);
      expect(state.workflowNotFound).toBe(false);
      expect(state.loading).toBe(false);
      expect(state.resolvedAttributes).toContain("attr1");
      expect(state.resolvedAttributes).toContain("result");
    });

    test("loadWorkflowData handles any error", async () => {
      mockApi.getWorkflowWithEvents.mockRejectedValue(
        new Error("Network error")
      );

      await useWorkflowStore.getState().loadWorkflowData("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.workflowNotFound).toBe(true);
      expect(state.loading).toBe(false);
      expect(state.workflowData).toBeNull();
      expect(state.executions).toEqual([]);
    });

    test("loadWorkflowData calculates resolved attributes from state and executions", async () => {
      const mockExecutions = [
        {
          step_id: "step-1",
          status: "completed",
          outputs: { attr2: "value2" },
        },
        { step_id: "step-2", status: "active" },
      ];

      mockApi.getWorkflowWithEvents.mockResolvedValue({
        workflow: mockWorkflow,
        executions: mockExecutions,
      });

      await useWorkflowStore.getState().loadWorkflowData("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.resolvedAttributes).toContain("attr1");
      expect(state.resolvedAttributes).toContain("attr2");
      expect(state.resolvedAttributes).toHaveLength(2);
    });
  });

  describe("Execution refresh", () => {
    test("refreshExecutions updates executions", async () => {
      const mockExecutions = [
        {
          step_id: "step-1",
          status: "completed",
          outputs: { result: "value" },
        },
      ];

      mockApi.getExecutions.mockResolvedValue(mockExecutions);

      await useWorkflowStore.getState().refreshExecutions("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.executions).toEqual(mockExecutions);
    });

    test("refreshExecutions handles error", async () => {
      mockApi.getExecutions.mockRejectedValue(new Error("Network error"));

      await useWorkflowStore.getState().refreshExecutions("wf-1");
      const state = useWorkflowStore.getState();

      expect(state.executions).toEqual([]);
    });
  });

  describe("Step health updates", () => {
    test("updateStepHealth updates step health", () => {
      useWorkflowStore.setState({ stepHealth: {} });

      useWorkflowStore.getState().updateStepHealth("step-1", "healthy");
      const state = useWorkflowStore.getState();

      expect(state.stepHealth["step-1"]).toEqual({ status: "healthy" });
    });

    test("updateStepHealth updates with error", () => {
      useWorkflowStore
        .getState()
        .updateStepHealth("step-1", "unhealthy", "Connection failed");
      const state = useWorkflowStore.getState();

      expect(state.stepHealth["step-1"]).toEqual({
        status: "unhealthy",
        error: "Connection failed",
      });
    });
  });

  describe("Workflow updates", () => {
    const mockWorkflow: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    test("updateWorkflowFromWebSocket merges state", () => {
      useWorkflowStore.setState({
        workflowData: mockWorkflow,
        workflows: [mockWorkflow],
        resolvedAttributes: [],
      });

      const update: Partial<WorkflowContext> = {
        status: "completed",
        state: { result: "final" },
      };

      useWorkflowStore.getState().updateWorkflowFromWebSocket(update);

      const state = useWorkflowStore.getState();

      expect(state.workflowData?.status).toBe("completed");
      expect(state.workflowData?.state).toEqual({ result: "final" });
      expect(state.resolvedAttributes).toContain("result");
    });

    test("updateWorkflowFromWebSocket updates workflows list", () => {
      const workflow2: WorkflowContext = {
        id: "wf-2",
        status: "active",
        state: {},
        started_at: "2024-01-02T00:00:00Z",
      };

      useWorkflowStore.setState({
        workflowData: mockWorkflow,
        workflows: [mockWorkflow, workflow2],
        resolvedAttributes: [],
      });

      const update: Partial<WorkflowContext> = {
        status: "completed",
      };

      useWorkflowStore.getState().updateWorkflowFromWebSocket(update);

      const state = useWorkflowStore.getState();
      const updatedWorkflow = state.workflows.find((w) => w.id === "wf-1");

      expect(updatedWorkflow?.status).toBe("completed");
    });

    test("updateWorkflowStatus updates workflow status", () => {
      useWorkflowStore.setState({
        workflowData: mockWorkflow,
        workflows: [mockWorkflow],
      });

      useWorkflowStore.getState().updateWorkflowStatus("wf-1", "completed");
      const state = useWorkflowStore.getState();

      expect(state.workflows[0].status).toBe("completed");
    });

    test("updateWorkflowStatus does nothing if workflow not found", () => {
      useWorkflowStore.setState({
        workflowData: mockWorkflow,
        workflows: [mockWorkflow],
      });

      useWorkflowStore.getState().updateWorkflowStatus("wf-999", "completed");
      const state = useWorkflowStore.getState();

      expect(state.workflows[0].status).toBe("active");
    });

    test("updateWorkflowStatus does nothing if status unchanged", () => {
      useWorkflowStore.setState({
        workflows: [mockWorkflow],
      });

      useWorkflowStore.getState().updateWorkflowStatus("wf-1", "active");
      const state = useWorkflowStore.getState();

      expect(state.workflows[0]).toEqual(mockWorkflow);
    });
  });

  describe("Selector hooks", () => {
    test("useSteps selector works", () => {
      const mockStep: Step = {
        id: "step-1",
        name: "Test",
        type: "sync",
        attributes: {},

        version: "1.0.0",
        http: {
          endpoint: "http://localhost:8080/test",
          timeout: 5000,
        },
      };

      useWorkflowStore.setState({ steps: [mockStep] });
      const { result } = renderHook(() => useSteps());
      expect(result.current).toEqual([mockStep]);
    });

    test("useWorkflows selector works", () => {
      const mockWorkflow: WorkflowContext = {
        id: "wf-1",
        status: "active",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
      };

      useWorkflowStore.setState({ workflows: [mockWorkflow] });
      const { result } = renderHook(() => useWorkflows());
      expect(result.current).toEqual([mockWorkflow]);
    });

    test("useSelectedWorkflow selector works", () => {
      useWorkflowStore.setState({ selectedWorkflow: "wf-1" });
      const { result } = renderHook(() => useSelectedWorkflow());
      expect(result.current).toBe("wf-1");
    });

    test("useWorkflowData selector works", () => {
      const mockWorkflow: WorkflowContext = {
        id: "wf-1",
        status: "active",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
      };
      useWorkflowStore.setState({ workflowData: mockWorkflow });
      const { result } = renderHook(() => useWorkflowData());
      expect(result.current).toEqual(mockWorkflow);
    });

    test("useExecutions selector works", () => {
      const executions = [{ step_id: "step-1", status: "completed" }];
      useWorkflowStore.setState({ executions });
      const { result } = renderHook(() => useExecutions());
      expect(result.current).toEqual(executions);
    });

    test("useResolvedAttributes selector works", () => {
      const attrs = ["attr1", "attr2"];
      useWorkflowStore.setState({ resolvedAttributes: attrs });
      const { result } = renderHook(() => useResolvedAttributes());
      expect(result.current).toEqual(attrs);
    });

    test("useWorkflowLoading selector works", () => {
      useWorkflowStore.setState({ loading: true });
      const { result } = renderHook(() => useWorkflowLoading());
      expect(result.current).toBe(true);
    });

    test("useWorkflowError selector works", () => {
      useWorkflowStore.setState({ error: "Test error" });
      const { result } = renderHook(() => useWorkflowError());
      expect(result.current).toBe("Test error");
    });

    test("useIsWorkflowMode selector works", () => {
      useWorkflowStore.setState({ isWorkflowMode: true });
      const { result } = renderHook(() => useIsWorkflowMode());
      expect(result.current).toBe(true);
    });
  });
});
