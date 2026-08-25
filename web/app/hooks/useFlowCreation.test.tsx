import { act, renderHook } from "@testing-library/react";
import { useFlowCreation } from "./useFlowCreation";
import { api, Step } from "../api";
import { snapshotFlowPositions } from "@/utils/nodePositioning";

const mockNavigate = jest.fn();

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => mockNavigate,
}));

const mockSteps: Step[] = [
  {
    id: "goal",
    name: "Goal Step",
    type: "script",
    attributes: {},
    script: { language: "lua", script: "" },
  },
];

const loadFlows = jest.fn().mockResolvedValue(undefined);
const addFlow = jest.fn();
const removeFlow = jest.fn();

jest.mock("../store/flowStore", () => ({
  useSteps: jest.fn(() => mockSteps),
  useLoadFlows: jest.fn(() => loadFlows),
  useAddFlow: jest.fn(() => addFlow),
  useRemoveFlow: jest.fn(() => removeFlow),
}));

let goalIds: string[] = [];
const uiState = {
  setPreviewPlan: jest.fn(),
  updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
  clearPreviewPlan: jest.fn(),
  get goalSteps() {
    return goalIds;
  },
  setGoalSteps: jest.fn((ids: string[]) => {
    goalIds = ids;
  }),
};

jest.mock("../contexts/UIContext", () => ({
  useUI: () => uiState,
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock("../api", () => ({
  api: {
    getExecutionPlan: jest.fn().mockResolvedValue({
      steps: { goal: {} },
      required: [],
    }),
    startFlow: jest.fn().mockResolvedValue(undefined),
  },
}));

jest.mock("@/utils/nodePositioning", () => ({
  snapshotFlowPositions: jest.fn(),
}));

describe("useFlowCreation", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    goalIds = [];
  });

  test("handles step change and sets derived flow id", async () => {
    const { result } = renderHook(() => useFlowCreation());

    await act(async () => {
      result.current.handleStepChange(["goal"]);
    });

    expect(uiState.setGoalSteps).toHaveBeenCalledWith(["goal"]);
    expect(uiState.updatePreviewPlan).toHaveBeenCalled();
    await act(async () => {});
    expect(result.current.newID).toMatch(/goal-step-/);
  });

  test("handles empty step change and clears preview", async () => {
    const { result } = renderHook(() => useFlowCreation());

    await act(async () => {
      result.current.handleStepChange([]);
    });

    expect(uiState.clearPreviewPlan).toHaveBeenCalled();
    expect(uiState.setGoalSteps).toHaveBeenCalledWith([]);
  });

  test("creates flow successfully and reloads flows", async () => {
    const { result } = renderHook(() => useFlowCreation());
    await act(async () => {
      await result.current.handleStepChange(["goal"]);
    });
    await act(async () => {
      result.current.setIDManuallyEdited(true);
      result.current.setNewID("flow-1");
    });

    await act(async () => {
      result.current.handleCreateFlow();
    });

    expect(snapshotFlowPositions).toHaveBeenCalledWith("flow-1", {
      type: "overview",
    });
    expect(addFlow).toHaveBeenCalled();
    expect(loadFlows).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith("/flow/flow-1");
    expect(uiState.clearPreviewPlan).toHaveBeenCalled();
  });

  test("removes optimistic flow on create error", async () => {
    (api.startFlow as jest.Mock).mockRejectedValueOnce(new Error("boom"));
    const { result } = renderHook(() => useFlowCreation());
    await act(async () => {
      await result.current.handleStepChange(["goal"]);
    });
    await act(async () => {
      result.current.setIDManuallyEdited(true);
      result.current.setNewID("flow-err");
    });

    await act(async () => {
      result.current.handleCreateFlow();
    });

    expect(removeFlow).toHaveBeenCalledWith("flow-err");
    expect(mockNavigate).toHaveBeenCalledWith("/");
  });
});
