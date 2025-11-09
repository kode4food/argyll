import axios from "axios";
import { SpudsApi } from "./client";
import type { WorkflowProjection } from "./types";
import { AttributeRole, AttributeType } from "./types";

jest.mock("axios");
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe("SpudsApi", () => {
  let api: SpudsApi;
  let mockClient: any;

  beforeEach(() => {
    mockClient = {
      get: jest.fn(),
      post: jest.fn(),
      put: jest.fn(),
      delete: jest.fn(),
    };
    mockedAxios.create.mockReturnValue(mockClient);
    api = new SpudsApi("http://localhost:8080");
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe("constructor", () => {
    test("creates axios client with default config", () => {
      expect(mockedAxios.create).toHaveBeenCalledWith({
        baseURL: "http://localhost:8080",
        timeout: 30000,
        headers: {
          "Content-Type": "application/json",
        },
      });
    });
  });

  describe("updateStep", () => {
    test("updates step and returns updated step", async () => {
      const mockStep = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
        attributes: {},
        version: "1.0.0",
        http: {
          endpoint: "http://localhost:8080/test",
          timeout: 5000,
        },
      };

      mockClient.put.mockResolvedValue({
        data: { step: mockStep },
      });

      const result = await api.updateStep("step-1", mockStep);

      expect(mockClient.put).toHaveBeenCalledWith(
        "/engine/step/step-1",
        mockStep
      );
      expect(result).toEqual(mockStep);
    });
  });

  describe("startWorkflow", () => {
    test("starts workflow with correct parameters", async () => {
      const mockResponse = { flow_id: "wf-1" };
      mockClient.post.mockResolvedValue({ data: mockResponse });

      const result = await api.startWorkflow("wf-1", ["step-1"], {
        input: "value",
      });

      expect(mockClient.post).toHaveBeenCalledWith("/engine/workflow", {
        id: "wf-1",
        goals: ["step-1"],
        init: { input: "value" },
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("getWorkflowWithEvents", () => {
    test("fetches workflow and converts projection", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "active",
        attributes: { result: { value: "value" } },
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: ["step-1"],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {
          "step-1": {
            status: "completed",
            inputs: { input: "value" },
            outputs: { result: "output" },
            started_at: "2024-01-01T00:00:00Z",
            completed_at: "2024-01-01T00:00:05Z",
            duration: 5000,
          },
        },
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getWorkflowWithEvents("wf-1");

      expect(mockClient.get).toHaveBeenCalledWith("/engine/workflow/wf-1");
      expect(result.workflow.id).toBe("wf-1");
      expect(result.workflow.status).toBe("active");
      expect(result.workflow.state).toEqual({ result: { value: "value" } });
      expect(result.executions).toHaveLength(1);
      expect(result.executions[0].step_id).toBe("step-1");
      expect(result.executions[0].status).toBe("completed");
    });

    test("handles workflow with error", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "failed",
        error: "Step execution failed",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: [],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {},
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getWorkflowWithEvents("wf-1");

      expect(result.workflow.error_state).toBeDefined();
      expect(result.workflow.error_state?.message).toBe(
        "Step execution failed"
      );
    });

    test("handles workflow with execution plan", async () => {
      const step1 = {
        id: "step-1",
        name: "Step 1",
        type: "sync" as const,
        attributes: {
          input1: {
            role: AttributeRole.Output,
            type: AttributeType.String,
          },
        },
        version: "1.0.0",
        http: {
          endpoint: "http://localhost:8080/test",
          timeout: 5000,
        },
      };

      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "active",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: ["step-2"],
          required: ["input1"],
          steps: {
            "step-1": { step: step1 },
          },
          attributes: {},
        },
        executions: {},
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getWorkflowWithEvents("wf-1");

      expect(result.workflow.plan).toBeDefined();
      expect(result.workflow.plan?.goals).toEqual(["step-2"]);
      expect(result.workflow.plan?.required).toEqual(["input1"]);
      expect(
        Object.keys(result.workflow.plan?.steps || {})
      ).toHaveLength(1);
    });

    test("handles empty execution plan", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "active",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: [],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {},
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getWorkflowWithEvents("wf-1");

      expect(result.workflow.plan).toBeUndefined();
    });
  });

  describe("listWorkflows", () => {
    test("fetches and converts workflow list", async () => {
      const mockProjections: WorkflowProjection[] = [
        {
          id: "wf-1",
          status: "active",
          attributes: {},
          created_at: "2024-01-01T00:00:00Z",
          plan: {
            goals: [],
            required: [],
            steps: {},
            attributes: {},
          },
          executions: {},
        },
        {
          id: "wf-2",
          status: "completed",
          attributes: { result: { value: "done" } },
          created_at: "2024-01-02T00:00:00Z",
          completed_at: "2024-01-02T00:05:00Z",
          plan: {
            goals: [],
            required: [],
            steps: {},
            attributes: {},
          },
          executions: {},
        },
      ];

      mockClient.get.mockResolvedValue({
        data: { workflows: mockProjections },
      });

      const result = await api.listWorkflows();

      expect(mockClient.get).toHaveBeenCalledWith("/engine/workflow");
      expect(result).toHaveLength(2);
      expect(result[0].id).toBe("wf-1");
      expect(result[1].id).toBe("wf-2");
      expect(result[1].completed_at).toBe("2024-01-02T00:05:00Z");
    });

    test("handles empty workflow list", async () => {
      mockClient.get.mockResolvedValue({ data: {} });

      const result = await api.listWorkflows();

      expect(result).toEqual([]);
    });
  });

  describe("getExecutions", () => {
    test("extracts executions from workflow", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "active",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: ["step-2"],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {
          "step-1": {
            status: "completed",
            inputs: { input: "value" },
            outputs: { result: "output" },
            started_at: "2024-01-01T00:00:00Z",
            completed_at: "2024-01-01T00:00:05Z",
            duration: 5000,
          },
          "step-2": {
            status: "active",
            inputs: { input: "output" },
            started_at: "2024-01-01T00:00:05Z",
          },
        },
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getExecutions("wf-1");

      expect(result).toHaveLength(2);
      expect(result[0].step_id).toBe("step-1");
      expect(result[0].status).toBe("completed");
      expect(result[0].flow_id).toBe("wf-1");
      expect(result[1].step_id).toBe("step-2");
      expect(result[1].status).toBe("active");
    });

    test("handles workflow with no executions", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "pending",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: [],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {},
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getExecutions("wf-1");

      expect(result).toEqual([]);
    });

    test("maps all execution statuses correctly", async () => {
      const mockProjection: WorkflowProjection = {
        id: "wf-1",
        status: "active",
        attributes: {},
        created_at: "2024-01-01T00:00:00Z",
        plan: {
          goals: [],
          required: [],
          steps: {},
          attributes: {},
        },
        executions: {
          "step-1": {
            status: "pending",
            inputs: {},
            started_at: "2024-01-01T00:00:00Z",
          },
          "step-2": {
            status: "active",
            inputs: {},
            started_at: "2024-01-01T00:00:01Z",
          },
          "step-3": {
            status: "completed",
            inputs: {},
            started_at: "2024-01-01T00:00:02Z",
          },
          "step-4": {
            status: "failed",
            inputs: {},
            started_at: "2024-01-01T00:00:03Z",
            error: "Error message",
          },
          "step-5": {
            status: "skipped",
            inputs: {},
            started_at: "2024-01-01T00:00:04Z",
          },
        },
      };

      mockClient.get.mockResolvedValue({ data: mockProjection });

      const result = await api.getExecutions("wf-1");

      expect(result).toHaveLength(5);
      expect(result[0].status).toBe("pending");
      expect(result[1].status).toBe("active");
      expect(result[2].status).toBe("completed");
      expect(result[3].status).toBe("failed");
      expect(result[3].error_message).toBe("Error message");
      expect(result[4].status).toBe("skipped");
    });
  });

  describe("getExecutionPlan", () => {
    test("fetches execution plan with goal steps", async () => {
      const step1 = {
        id: "step-1",
        name: "Step 1",
        type: "sync" as const,
        attributes: {
          input1: {
            role: AttributeRole.Output,
            type: AttributeType.String,
          },
        },
        version: "1.0.0",
        http: {
          endpoint: "http://localhost:8080/test",
          timeout: 5000,
        },
      };

      const mockPlan = {
        goals: ["step-2"],
        required: ["input1"],
        steps: {
          "step-1": { step: step1 },
        },
        attributes: {},
      };

      mockClient.post.mockResolvedValue({ data: mockPlan });

      const result = await api.getExecutionPlan(["step-2"], { input: "value" });

      expect(mockClient.post).toHaveBeenCalledWith(
        "/engine/plan",
        {
          goals: ["step-2"],
          init: { input: "value" },
        },
        {
          signal: undefined,
        }
      );
      expect(result).toEqual(mockPlan);
    });

    test("fetches execution plan with empty initial state", async () => {
      const mockPlan = {
        goals: ["step-1"],
        required: [],
        steps: {},
      };

      mockClient.post.mockResolvedValue({ data: mockPlan });

      const result = await api.getExecutionPlan(["step-1"]);

      expect(mockClient.post).toHaveBeenCalledWith(
        "/engine/plan",
        {
          goals: ["step-1"],
          init: {},
        },
        {
          signal: undefined,
        }
      );
      expect(result).toEqual(mockPlan);
    });
  });

  describe("getEngineState", () => {
    test("fetches engine state with steps and health", async () => {
      const mockState = {
        steps: {
          "step-1": {
            id: "step-1",
            name: "Step 1",
            type: "sync",
            required: {},
            optional: {},
            output: {},
            version: "1.0.0",
            http: {
              endpoint: "http://localhost:8080/test",
              timeout: 5000,
            },
          },
        },
        health: {
          "step-1": { status: "healthy" },
        },
      };

      mockClient.get.mockResolvedValue({ data: mockState });

      const result = await api.getEngineState();

      expect(mockClient.get).toHaveBeenCalledWith("/engine");
      expect(result.steps).toEqual(mockState.steps);
      expect(result.health).toEqual(mockState.health);
    });
  });
});
