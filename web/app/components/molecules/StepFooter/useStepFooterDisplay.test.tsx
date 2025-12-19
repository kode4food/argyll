import { renderHook } from "@testing-library/react";
import { useStepFooterDisplay } from "./useStepFooterDisplay";
import { ExecutionResult } from "../../../api";

describe("useStepFooterDisplay", () => {
  const mockProgressState = {
    status: "pending" as const,
    flowId: "flow-1",
  };

  const mockProgressStateWithWorkItems = {
    status: "active" as const,
    flowId: "flow-1",
    workItems: {
      total: 5,
      completed: 2,
      failed: 1,
      active: 2,
    },
  };

  const mockHealthText = "All systems healthy";

  describe("display info", () => {
    it("returns script display info for script steps", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "script" as const,
        script: {
          language: "python",
          script: "print('hello')\nprint('world')",
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      expect(result.current.displayInfo).not.toBeNull();
      expect(result.current.displayInfo?.text).toBe(
        "print('hello') print('world')"
      );
    });

    it("returns http display info for http steps", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
        http: {
          endpoint: "https://api.example.com/users",
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      expect(result.current.displayInfo).not.toBeNull();
      expect(result.current.displayInfo?.text).toBe(
        "https://api.example.com/users"
      );
    });

    it("returns null for steps with no script or http", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      expect(result.current.displayInfo).toBeNull();
    });
  });

  describe("tooltip sections", () => {
    it("includes execution status section when execution and flowId are present", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "completed",
        duration_ms: 1000,
        error_message: null,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      expect(result.current.tooltipSections.length).toBeGreaterThan(0);
      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("execution-status");
    });

    it("includes error section for failed execution", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "failed",
        error_message: "Connection timeout",
        duration_ms: 500,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("error");
    });

    it("includes skip reason for skipped execution with predicate", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "sync" as const,
        predicate: "x > 0",
      };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "skipped",
        duration_ms: 0,
        error_message: null,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("reason");
    });

    it("includes duration for completed execution", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "completed",
        duration_ms: 5000,
        error_message: null,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("duration");
    });

    it("includes execution status with work items when workItems present", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "active",
        duration_ms: null,
        error_message: null,
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          mockProgressStateWithWorkItems
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("execution-status");
    });

    it("includes execution status when workItems has total of 1", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };
      const execution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "active",
        duration_ms: null,
        error_message: null,
      };

      const progressStateSingleItem = {
        status: "active" as const,
        flowId: "flow-1",
        workItems: {
          total: 1,
          completed: 0,
          failed: 0,
          active: 1,
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          execution,
          "flow-1",
          "healthy",
          undefined,
          mockHealthText,
          progressStateSingleItem
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("execution-status");
    });

    it("includes script preview when no flowId and step is script", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "script" as const,
        script: {
          language: "python",
          script: "line1\nline2\nline3",
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("script");
    });

    it("includes script preview for longer scripts", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "script" as const,
        script: {
          language: "python",
          script: "line1\nline2\nline3\nline4\nline5\nline6\nline7",
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("script");
    });

    it("includes script preview for shorter scripts", () => {
      const step = {
        id: "step-1",
        name: "Test Step",
        type: "script" as const,
        script: {
          language: "python",
          script: "line1\nline2\nline3\nline4\nline5",
        },
      };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("script");
    });

    it("includes health status section when no flowId", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };

      const { result } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined, // no flowId
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const keys = result.current.tooltipSections.map((s) => s.key);
      expect(keys).toContain("health");
    });
  });

  describe("memoization", () => {
    it("memoizes results and doesn't recompute on unchanged props", () => {
      const step = { id: "step-1", name: "Test Step", type: "sync" as const };

      const { result, rerender } = renderHook(() =>
        useStepFooterDisplay(
          step as any,
          undefined,
          undefined,
          "healthy",
          undefined,
          mockHealthText,
          mockProgressState
        )
      );

      const firstResult = result.current;

      rerender();

      expect(result.current).toBe(firstResult);
    });

    it("recomputes when step changes", () => {
      const step1 = { id: "step-1", name: "Test Step", type: "sync" as const };
      const step2 = {
        id: "step-2",
        name: "Test Step 2",
        type: "sync" as const,
      };

      const { result, rerender } = renderHook(
        ({ step }) =>
          useStepFooterDisplay(
            step as any,
            undefined,
            undefined,
            "healthy",
            undefined,
            mockHealthText,
            mockProgressState
          ),
        { initialProps: { step: step1 } }
      );

      const firstResult = result.current;

      rerender({ step: step2 });

      expect(result.current).not.toBe(firstResult);
    });
  });
});
