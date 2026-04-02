import { renderHook } from "@testing-library/react";
import { useStepHealth } from "./useStepHealth";
import { useFlowStore } from "../store/flowStore";
import type { Step } from "../api";

jest.mock("../store/flowStore");

describe("useStepHealth", () => {
  const mockUseFlowStore = useFlowStore as jest.MockedFunction<
    typeof useFlowStore
  >;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  const setupMock = (stepHealth: Record<string, any>) => {
    mockUseFlowStore.mockImplementation((selector: any) =>
      selector({ stepHealth })
    );
  };

  const syncStep = (hasHealthCheck: boolean): Step => ({
    id: "step-1",
    name: "Sync Step",
    type: "sync",
    attributes: {},
    http: {
      endpoint: "http://test.com",
      timeout: 5000,
      ...(hasHealthCheck && { health_check: "http://test.com/health" }),
    },
  });

  const asyncStep = (hasHealthCheck: boolean): Step => ({
    id: "step-1",
    name: "Async Step",
    type: "async",
    attributes: {},
    http: {
      endpoint: "http://test.com",
      timeout: 5000,
      ...(hasHealthCheck && { health_check: "http://test.com/health" }),
    },
  });

  const scriptStep: Step = {
    id: "step-script",
    name: "Script Step",
    type: "script",
    attributes: {},
    script: { language: "ale", script: "{}" },
  };

  const flowStep = (goals: string[]): Step => ({
    id: "step-flow",
    name: "Flow Step",
    type: "flow",
    attributes: {},
    flow: { goals },
  });

  describe("sync steps", () => {
    test("returns unconfigured without health check", () => {
      setupMock({});
      const { result } = renderHook(() => useStepHealth(syncStep(false)));
      expect(result.current.status).toBe("unconfigured");
    });

    test("returns health from store with health check", () => {
      setupMock({ "step-1": { status: "healthy" } });
      const { result } = renderHook(() => useStepHealth(syncStep(true)));
      expect(result.current.status).toBe("healthy");
    });

    test("returns unknown when no health info", () => {
      setupMock({});
      const { result } = renderHook(() => useStepHealth(syncStep(true)));
      expect(result.current.status).toBe("unknown");
    });

    test("returns per-node health", () => {
      setupMock({
        "step-1": {
          status: "unhealthy",
          error: "node node-2: Connection failed",
          nodes: {
            "node-1": { status: "healthy" },
            "node-2": { status: "unhealthy", error: "Connection failed" },
          },
        },
      });
      const { result } = renderHook(() => useStepHealth(syncStep(true)));
      expect(result.current.nodes).toEqual([
        { nodeId: "node-1", status: "healthy", error: undefined },
        { nodeId: "node-2", status: "unhealthy", error: "Connection failed" },
      ]);
    });
  });

  describe("async steps", () => {
    test("returns unconfigured without health check", () => {
      setupMock({});
      const { result } = renderHook(() => useStepHealth(asyncStep(false)));
      expect(result.current.status).toBe("unconfigured");
    });

    test("returns health from store with health check", () => {
      setupMock({
        "step-1": { status: "unhealthy", error: "Connection failed" },
      });
      const { result } = renderHook(() => useStepHealth(asyncStep(true)));
      expect(result.current.status).toBe("unhealthy");
      expect(result.current.error).toBe("Connection failed");
    });
  });

  describe("script steps", () => {
    test("returns health status from store", () => {
      setupMock({ "step-script": { status: "healthy" } });
      const { result } = renderHook(() => useStepHealth(scriptStep));
      expect(result.current.status).toBe("healthy");
    });

    test("does not return per-node health", () => {
      setupMock({
        "step-script": {
          status: "healthy",
          nodes: { "argyll-1": { status: "healthy" } },
        },
      });
      const { result } = renderHook(() => useStepHealth(scriptStep));
      expect(result.current.nodes).toBeUndefined();
    });

    test("returns unknown when no health info", () => {
      setupMock({});
      const { result } = renderHook(() => useStepHealth(scriptStep));
      expect(result.current.status).toBe("unknown");
    });
  });

  describe("flow steps", () => {
    test("returns unknown when no health info in store", () => {
      setupMock({});
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-step"]))
      );
      expect(result.current.status).toBe("unknown");
    });

    test("derives overall status from goal step status", () => {
      setupMock({ "goal-step": { status: "healthy" } });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-step"]))
      );
      expect(result.current.status).toBe("healthy");
    });

    test("derives overall status as worst-case across goals", () => {
      setupMock({
        "goal-a": { status: "healthy" },
        "goal-b": { status: "unhealthy", error: "down" },
      });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-a", "goal-b"]))
      );
      expect(result.current.status).toBe("unhealthy");
      expect(result.current.error).toBe("down");
    });

    test("returns no nodes when goal steps have no per-node data", () => {
      setupMock({ "goal-step": { status: "healthy" } });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-step"]))
      );
      expect(result.current.nodes).toBeUndefined();
    });

    test("derives per-node health from goal step nodes", () => {
      setupMock({
        "goal-step": {
          status: "healthy",
          nodes: {
            "node-1": { status: "healthy" },
            "node-2": { status: "healthy" },
          },
        },
      });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-step"]))
      );
      expect(result.current.nodes).toEqual([
        { nodeId: "node-1", status: "healthy", error: undefined },
        { nodeId: "node-2", status: "healthy", error: undefined },
      ]);
    });

    test("takes worst-case status per node across multiple goals", () => {
      setupMock({
        "goal-a": {
          status: "healthy",
          nodes: {
            "node-1": { status: "healthy" },
            "node-2": { status: "unhealthy", error: "timeout" },
          },
        },
        "goal-b": {
          status: "healthy",
          nodes: {
            "node-1": { status: "unknown" },
            "node-2": { status: "healthy" },
          },
        },
      });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-a", "goal-b"]))
      );
      expect(result.current.nodes).toEqual([
        { nodeId: "node-1", status: "unknown", error: undefined },
        { nodeId: "node-2", status: "unhealthy", error: "timeout" },
      ]);
    });

    test("includes nodes only from goals that have per-node data", () => {
      setupMock({
        "goal-script": { status: "healthy" },
        "goal-http": {
          status: "healthy",
          nodes: { "node-1": { status: "healthy" } },
        },
      });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-script", "goal-http"]))
      );
      expect(result.current.nodes).toEqual([
        { nodeId: "node-1", status: "healthy", error: undefined },
      ]);
    });

    test("returns nodes sorted by node ID", () => {
      setupMock({
        "goal-step": {
          status: "healthy",
          nodes: {
            "node-3": { status: "healthy" },
            "node-1": { status: "healthy" },
            "node-2": { status: "healthy" },
          },
        },
      });
      const { result } = renderHook(() =>
        useStepHealth(flowStep(["goal-step"]))
      );
      expect(result.current.nodes?.map((n) => n.nodeId)).toEqual([
        "node-1",
        "node-2",
        "node-3",
      ]);
    });
  });
});
