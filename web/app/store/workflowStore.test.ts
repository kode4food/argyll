import { renderHook } from "@testing-library/react";
import { useWorkflowStore, useSteps, useWorkflows } from "./workflowStore";
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

    test("addStep creates new step", () => {
      useWorkflowStore.getState().addStep(mockStep);
      expect(useWorkflowStore.getState().steps).toHaveLength(1);
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

    test("addWorkflow adds workflow", () => {
      useWorkflowStore.getState().addWorkflow(mockWorkflow);
      expect(useWorkflowStore.getState().workflows).toHaveLength(1);
    });

    test("removeWorkflow deletes workflow", () => {
      useWorkflowStore.setState({ workflows: [mockWorkflow] });
      useWorkflowStore.getState().removeWorkflow("wf-1");
      expect(useWorkflowStore.getState().workflows).toHaveLength(0);
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
  });
});
