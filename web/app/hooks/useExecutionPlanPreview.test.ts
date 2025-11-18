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
      setGoalStepIds: jest.fn(),
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("returns initial state with null preview plan", () => {
    const onSelectStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(null, onSelectStep)
    );

    expect(result.current.previewPlan).toBeNull();
    expect(typeof result.current.handleStepClick).toBe("function");
    expect(typeof result.current.clearPreview).toBe("function");
  });

  test("handleStepClick updates preview when no flow", async () => {
    const onSelectStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(null, onSelectStep)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockUpdatePreviewPlan).toHaveBeenCalledWith(["step-1"], {});
    expect(onSelectStep).toHaveBeenCalledWith("step-1");
  });

  test("handleStepClick does nothing when flow is active", async () => {
    const onSelectStep = jest.fn();
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { result } = renderHook(() =>
      useExecutionPlanPreview(null, onSelectStep, flowData)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
    expect(onSelectStep).not.toHaveBeenCalled();
  });

  test("handleStepClick clears preview when clicking same step", async () => {
    const onSelectStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview("step-1", onSelectStep)
    );

    await act(async () => {
      await result.current.handleStepClick("step-1");
    });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(onSelectStep).toHaveBeenCalledWith(null);
    expect(mockUpdatePreviewPlan).not.toHaveBeenCalled();
  });

  test("clearPreview clears plan and selection", () => {
    const onSelectStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview("step-1", onSelectStep)
    );

    act(() => {
      result.current.clearPreview();
    });

    expect(mockClearPreviewPlan).toHaveBeenCalled();
    expect(onSelectStep).toHaveBeenCalledWith(null);
  });

  test("clears preview when flow becomes active", () => {
    const onSelectStep = jest.fn();
    const { rerender } = renderHook(
      ({ flowData }) =>
        useExecutionPlanPreview("step-1", onSelectStep, flowData),
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
      selectedStep: null,
      setSelectedStep: jest.fn(),
      goalStepIds: [],
      setGoalStepIds: jest.fn(),
    });

    const onSelectStep = jest.fn();
    const { result } = renderHook(() =>
      useExecutionPlanPreview(null, onSelectStep)
    );

    expect(result.current.previewPlan).toEqual(mockPlan);
  });
});
