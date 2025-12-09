import { renderHook, act } from "@testing-library/react";
import { useExecutionPlanPreview } from "./useExecutionPlanPreview";
import { useUI } from "../contexts/UIContext";
import type { FlowContext, ExecutionPlan } from "../api";

jest.mock("../contexts/UIContext");
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
      selectedStep: null,
      setSelectedStep: jest.fn(),
      goalStepIds: [],
      toggleGoalStep: jest.fn(),
      setGoalStepIds: jest.fn(),
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("returns initial state with null preview plan", () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], onSelectStep, onToggleStep)
    );

    expect(result.current.previewPlan).toBeNull();
    expect(typeof result.current.handleStepClick).toBe("function");
    expect(typeof result.current.clearPreview).toBe("function");
  });

  test("handleStepClick updates preview when no flow", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], onSelectStep, onToggleStep)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(["step-1"], {});
    expect(onSelectStep).toHaveBeenCalledWith("step-1");
    expect(onToggleStep).not.toHaveBeenCalled();
  });

  test("handleStepClick does nothing when flow is active", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useExecutionPlanPreview([], onSelectStep, onToggleStep, flowData)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
    expect(onSelectStep).not.toHaveBeenCalled();
    expect(onToggleStep).not.toHaveBeenCalled();
  });

  test("handleStepClick clears preview when clicking same step", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], onSelectStep, onToggleStep)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(onSelectStep).toHaveBeenCalledWith(null);
    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
    expect(onToggleStep).not.toHaveBeenCalled();
  });

  test("handleStepClick toggles selection additively", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], onSelectStep, onToggleStep)
    );

    await act(async () => {
      await result.current.handleStepClick("step-2", { additive: true });
    });

    expect(onToggleStep).toHaveBeenCalledWith("step-2");
    expect(onSelectStep).not.toHaveBeenCalled();
    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(
      ["step-1", "step-2"],
      {}
    );
  });

  test("additive click ignores steps already in preview plan", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();

    mockUseUI.mockReturnValueOnce({
      previewPlan: {
        steps: { "in-plan": {} as any },
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
      selectedStep: null,
      setSelectedStep: jest.fn(),
      goalStepIds: ["goal"],
      toggleGoalStep: jest.fn(),
      setGoalStepIds: jest.fn(),
    });

    const { result } = renderHook(() =>
      useExecutionPlanPreview(["goal"], onSelectStep, onToggleStep)
    );

    await act(async () => {
      await result.current.handleStepClick("in-plan", { additive: true });
    });

    expect(onToggleStep).not.toHaveBeenCalled();
    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
    expect(onSelectStep).not.toHaveBeenCalled();
  });

  test("normal click still replaces selection when step already in plan", async () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();

    mockUseUI.mockReturnValueOnce({
      previewPlan: {
        steps: { "blocked-step": {} as any },
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
      selectedStep: null,
      setSelectedStep: jest.fn(),
      goalStepIds: ["goal"],
      toggleGoalStep: jest.fn(),
      setGoalStepIds: jest.fn(),
    });

    const { result } = renderHook(() =>
      useExecutionPlanPreview(["goal"], onSelectStep, onToggleStep)
    );

    await act(async () => {
      await result.current.handleStepClick("blocked-step");
    });

    expect(onToggleStep).not.toHaveBeenCalled();
    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(["blocked-step"], {});
    expect(onSelectStep).toHaveBeenCalledWith("blocked-step");
  });

  test("clearPreview clears plan and selection", () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(["step-1"], onSelectStep, onToggleStep)
    );

    act(() => {
      result.current.clearPreview();
    });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(onSelectStep).toHaveBeenCalledWith(null);
    expect(onToggleStep).not.toHaveBeenCalled();
  });

  test("clears preview when flow becomes active", () => {
    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { rerender } = renderHook(
      ({ flowData }) =>
        useExecutionPlanPreview(
          ["step-1"],
          onSelectStep,
          onToggleStep,
          flowData
        ),
      {
        initialProps: { flowData: null as FlowContext | null },
      }
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
      selectedStep: null,
      setSelectedStep: jest.fn(),
      goalStepIds: [],
      toggleGoalStep: jest.fn(),
      setGoalStepIds: jest.fn(),
    });

    const onSelectStep = jest.fn();
    const onToggleStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview([], onSelectStep, onToggleStep)
    );

    expect(result.current.previewPlan).toEqual(mockPlan);
  });
});
