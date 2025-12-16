import { renderHook, act } from "@testing-library/react";
import { useExecutionPlanPreview } from "./useExecutionPlanPreview";
import { useUI } from "@/app/contexts/UIContext";
import type { FlowContext, ExecutionPlan, Step } from "@/app/api";

jest.mock("@/app/contexts/UIContext");
const mockUseUI = useUI as jest.MockedFunction<typeof useUI>;

describe("useExecutionPlanPreview", () => {
  let mockUpdatePreviewPlan: jest.Mock;
  let mockClearPreviewPlan: jest.Mock;

  beforeEach(() => {
    mockUpdatePreviewPlan = jest.fn();
    mockClearPreviewPlan = jest.fn();

    mockUseUI.mockReturnValue({
      previewPlan: null,
      updatePreviewPlan: mockUpdatePreviewPlan,
      clearPreviewPlan: mockClearPreviewPlan,
      showCreateForm: false,
      setShowCreateForm: jest.fn(),
      disableEdit: false,
      diagramContainerRef: { current: null },
      goalSteps: [],
      toggleGoalStep: jest.fn(),
      setGoalSteps: jest.fn(),
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("returns initial state with null preview plan", () => {
    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], setGoalSteps)
    );

    expect(result.current.previewPlan).toBeNull();
    expect(typeof result.current.handleStepClick).toBe("function");
    expect(typeof result.current.clearPreview).toBe("function");
  });

  test("handleStepClick updates preview when no flow", async () => {
    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], setGoalSteps)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(setGoalSteps).toHaveBeenCalledWith(["step-1"]);
    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(["step-1"], {});
  });

  test("handleStepClick does nothing when flow is active", async () => {
    const setGoalSteps = jest.fn();
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useExecutionPlanPreview([], setGoalSteps, flowData)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
    expect(setGoalSteps).not.toHaveBeenCalled();
  });

  test("handleStepClick clears preview when clicking same step", async () => {
    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], setGoalSteps)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(setGoalSteps).toHaveBeenCalledWith([]);
    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
  });

  test("additive click toggles selection", async () => {
    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], setGoalSteps)
    );

    await act(async () => {
      await result.current.handleStepClick("step-2", { additive: true });
    });

    expect(setGoalSteps).toHaveBeenCalledWith(["step-1", "step-2"]);
    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(
      ["step-1", "step-2"],
      {}
    );
  });

  test("additive click ignores steps already in preview plan", async () => {
    const setGoalSteps = jest.fn();

    mockUseUI.mockReturnValueOnce({
      previewPlan: {
        steps: { "in-plan": {} as Step },
        goals: ["goal"],
        required: [],
        attributes: {},
      } as ExecutionPlan,
      updatePreviewPlan: mockUpdatePreviewPlan,
      clearPreviewPlan: mockClearPreviewPlan,
      showCreateForm: false,
      setShowCreateForm: jest.fn(),
      disableEdit: false,
      diagramContainerRef: { current: null },
      goalSteps: ["goal"],
      toggleGoalStep: jest.fn(),
      setGoalSteps: jest.fn(),
    });

    const { result } = renderHook(() =>
      useExecutionPlanPreview(["goal"], setGoalSteps)
    );

    await act(async () => {
      await result.current.handleStepClick("in-plan", { additive: true });
    });

    expect(setGoalSteps).not.toHaveBeenCalled();
    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
  });

  test("normal click still replaces selection when step already in plan", async () => {
    const setGoalSteps = jest.fn();

    mockUseUI.mockReturnValueOnce({
      previewPlan: {
        steps: { "blocked-step": {} as Step },
        goals: ["goal"],
        attributes: {},
        required: [],
      } as ExecutionPlan,
      updatePreviewPlan: mockUpdatePreviewPlan,
      clearPreviewPlan: mockClearPreviewPlan,
      showCreateForm: false,
      setShowCreateForm: jest.fn(),
      disableEdit: false,
      diagramContainerRef: { current: null },
      goalSteps: ["goal"],
      toggleGoalStep: jest.fn(),
      setGoalSteps: jest.fn(),
    });

    const { result } = renderHook(() =>
      useExecutionPlanPreview(["goal"], setGoalSteps)
    );

    await act(async () => {
      await result.current.handleStepClick("blocked-step");
    });

    expect(setGoalSteps).toHaveBeenCalledWith(["blocked-step"]);
    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(["blocked-step"], {});
  });

  test("clearPreview clears plan and selection", () => {
    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], setGoalSteps)
    );

    act(() => {
      result.current.clearPreview();
    });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(setGoalSteps).toHaveBeenCalledWith([]);
  });

  test("clears preview when flow becomes active", () => {
    const setGoalSteps = jest.fn();
    const { rerender } = renderHook(
      ({ flowData }) =>
        useExecutionPlanPreview(["step-1"], setGoalSteps, flowData),
      { initialProps: { flowData: null as FlowContext | null } }
    );

    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    rerender({ flowData });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
  });

  test("returns preview plan from context", () => {
    const mockPlan: ExecutionPlan = {
      goals: ["step-1"],
      required: [],
      steps: {},
      attributes: {},
    };

    mockUseUI.mockReturnValue({
      previewPlan: mockPlan,
      updatePreviewPlan: mockUpdatePreviewPlan,
      clearPreviewPlan: mockClearPreviewPlan,
      showCreateForm: false,
      setShowCreateForm: jest.fn(),
      disableEdit: false,
      diagramContainerRef: { current: null },
      goalSteps: [],
      toggleGoalStep: jest.fn(),
      setGoalSteps: jest.fn(),
    });

    const setGoalSteps = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], setGoalSteps)
    );

    expect(result.current.previewPlan).toEqual(mockPlan);
  });
});
