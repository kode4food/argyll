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

  const createStep = (
    type: "sync" | "async" | "script",
    hasHealthCheck: boolean
  ): Step => ({
    id: "test-step",
    name: "Test Step",
    type,
    attributes: {},

    version: "1.0.0",
    ...(type !== "script" && {
      http: {
        endpoint: "http://test.com",
        timeout: 5000,
        ...(hasHealthCheck && { health_check: "http://test.com/health" }),
      },
    }),
  });

  test("returns unconfigured for sync step without health check", () => {
    mockUseFlowStore.mockReturnValue({});
    const step = createStep("sync", false);
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("unconfigured");
  });

  test("returns unconfigured for async step without health check", () => {
    mockUseFlowStore.mockReturnValue({});
    const step = createStep("async", false);
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("unconfigured");
  });

  test("returns health from store for sync step with health check", () => {
    mockUseFlowStore.mockReturnValue({
      status: "healthy",
    });
    const step = createStep("sync", true);
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("healthy");
  });

  test("returns health from store for async step with health check", () => {
    mockUseFlowStore.mockReturnValue({
      status: "unhealthy",
      error: "Connection failed",
    });
    const step = createStep("async", true);
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("unhealthy");
    expect(result.current.error).toBe("Connection failed");
  });

  test("returns unknown status when no health info in store", () => {
    mockUseFlowStore.mockReturnValue({});
    const step = createStep("sync", true);
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("unknown");
  });

  test("returns health for script step", () => {
    mockUseFlowStore.mockReturnValue({
      status: "healthy",
    });
    const step: Step = {
      id: "script-step",
      name: "Script",
      type: "script",
      attributes: {},

      version: "1.0.0",
      script: {
        language: "ale",
        script: "{}",
      },
    };
    const { result } = renderHook(() => useStepHealth(step));
    expect(result.current.status).toBe("healthy");
  });
});
