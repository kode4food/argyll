import React from "react";
import { renderHook, act, waitFor } from "@testing-library/react";
import { UIProvider, useUI } from "./UIContext";
import { api, ExecutionPlan } from "../api";

jest.mock("../api", () => ({
  ...jest.requireActual("../api"),
  api: {
    getExecutionPlan: jest.fn(),
  },
}));

const mockApi = api as jest.Mocked<typeof api>;

describe("UIContext", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <UIProvider>{children}</UIProvider>
  );

  test("throws error when used outside provider", () => {
    const consoleSpy = jest.spyOn(console, "error");

    expect(() => {
      renderHook(() => useUI());
    }).toThrow("useUI must be used within a UIProvider");

    consoleSpy.mockRestore();
  });

  test("provides initial state", () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    expect(result.current.showCreateForm).toBe(false);
    expect(result.current.disableEdit).toBe(false);
    expect(result.current.previewPlan).toBeNull();
    expect(result.current.goalSteps).toEqual([]);
  });

  test("setShowCreateForm updates state", () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    act(() => {
      result.current.setShowCreateForm(true);
    });

    expect(result.current.showCreateForm).toBe(true);
    expect(result.current.disableEdit).toBe(true);
  });

  test("setGoalSteps updates state", () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    act(() => {
      result.current.setGoalSteps(["step-1", "step-2"]);
    });

    expect(result.current.goalSteps).toEqual(["step-1", "step-2"]);
  });

  test("toggleGoalStep adds and removes ids", () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    act(() => {
      result.current.toggleGoalStep("step-1");
    });
    expect(result.current.goalSteps).toEqual(["step-1"]);

    act(() => {
      result.current.toggleGoalStep("step-2");
    });
    expect(result.current.goalSteps).toEqual(["step-1", "step-2"]);

    act(() => {
      result.current.toggleGoalStep("step-2");
    });
    expect(result.current.goalSteps).toEqual(["step-1"]);
  });

  test("updatePreviewPlan calls API and updates state", async () => {
    const mockPlan = {
      steps: {},
      attributes: {},
      goals: ["step-1"],
      required: [],
    };

    mockApi.getExecutionPlan.mockResolvedValue(mockPlan);

    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan(["step-1"], { foo: "bar" });
    });

    expect(mockApi.getExecutionPlan).toHaveBeenCalledWith(
      ["step-1"],
      { foo: "bar" },
      expect.any(AbortSignal)
    );
    expect(result.current.previewPlan).toEqual(mockPlan);
  });

  test("updatePreviewPlan clears plan when goalSteps is empty", async () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan([], {});
    });

    expect(mockApi.getExecutionPlan).not.toHaveBeenCalled();
    expect(result.current.previewPlan).toBeNull();
  });

  test("updatePreviewPlan handles errors", async () => {
    const consoleErrorSpy = jest.spyOn(console, "error");
    mockApi.getExecutionPlan.mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan(["step-1"], {});
    });

    expect(consoleErrorSpy).toHaveBeenCalledWith(
      "Failed to update preview plan:",
      expect.any(Error)
    );
    expect(result.current.previewPlan).toBeNull();

    consoleErrorSpy.mockRestore();
  });

  test("updatePreviewPlan ignores abort errors", async () => {
    const consoleErrorSpy = jest.spyOn(console, "error");
    const abortError = new Error("Aborted");
    abortError.name = "AbortError";
    mockApi.getExecutionPlan.mockRejectedValue(abortError);

    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan(["step-1"], {});
    });

    expect(consoleErrorSpy).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });

  test("updatePreviewPlan ignores canceled errors", async () => {
    const consoleErrorSpy = jest.spyOn(console, "error");
    const cancelError = Object.assign(new Error("Canceled"), {
      code: "ERR_CANCELED",
    });
    mockApi.getExecutionPlan.mockRejectedValue(cancelError);

    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan(["step-1"], {});
    });

    expect(consoleErrorSpy).not.toHaveBeenCalled();

    consoleErrorSpy.mockRestore();
  });

  test("updatePreviewPlan aborts previous request", async () => {
    let resolveFirst: (value: ExecutionPlan) => void = () => {};
    const firstPromise = new Promise<ExecutionPlan>((resolve) => {
      resolveFirst = resolve;
    });

    mockApi.getExecutionPlan.mockImplementation(() => firstPromise);

    const { result } = renderHook(() => useUI(), { wrapper });

    // Start first request
    act(() => {
      result.current.updatePreviewPlan(["step-1"], {});
    });

    const mockPlan2 = {
      steps: {},
      attributes: {},
      goals: ["step-2"],
      required: [],
    } as ExecutionPlan;

    // Start second request (should abort first)
    mockApi.getExecutionPlan.mockResolvedValue(mockPlan2);
    await act(async () => {
      await result.current.updatePreviewPlan(["step-2"], {});
    });

    // Complete first request
    resolveFirst({
      steps: {},
      attributes: {},
      goals: ["step-1"],
      required: [],
    });

    await waitFor(() => {
      expect(result.current.previewPlan).toEqual(mockPlan2);
    });
  });

  test("clearPreviewPlan clears state and aborts request", async () => {
    const mockPlan = {
      steps: {},
      attributes: {},
      goals: ["step-1"],
      required: [],
    };

    mockApi.getExecutionPlan.mockResolvedValue(mockPlan);

    const { result } = renderHook(() => useUI(), { wrapper });

    await act(async () => {
      await result.current.updatePreviewPlan(["step-1"], {});
    });

    expect(result.current.previewPlan).toEqual(mockPlan);

    act(() => {
      result.current.clearPreviewPlan();
    });

    expect(result.current.previewPlan).toBeNull();
  });

  test("aborts pending request on unmount", async () => {
    let abortSignal: AbortSignal | undefined;
    mockApi.getExecutionPlan.mockImplementation(
      (_goals, _state, signal?: AbortSignal) => {
        abortSignal = signal;
        return new Promise(() => {}); // Never resolves
      }
    );

    const { result, unmount } = renderHook(() => useUI(), { wrapper });

    act(() => {
      result.current.updatePreviewPlan(["step-1"], {});
    });

    unmount();

    expect(abortSignal?.aborted).toBe(true);
  });

  test("diagramContainerRef is provided", () => {
    const { result } = renderHook(() => useUI(), { wrapper });

    expect(result.current.diagramContainerRef).toBeDefined();
    expect(result.current.diagramContainerRef.current).toBeNull();
  });
});
