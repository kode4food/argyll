import React from "react";
import { act, render } from "@testing-library/react";
import {
  FlowCreationStateProvider,
  useFlowCreation,
} from "./FlowCreationContext";
import { ExecutionPlan, Step } from "../api";
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
let previewPlan: ExecutionPlan | null = null;
const uiState = {
  previewPlan,
  setPreviewPlan: jest.fn(),
  updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
  clearPreviewPlan: jest.fn(),
  toggleGoalStep: jest.fn(),
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

const apiMock = require("../api").api;

let flowCtx: ReturnType<typeof useFlowCreation> | null = null;

const Consumer = () => {
  flowCtx = useFlowCreation();
  return null;
};

const renderProvider = () =>
  render(
    <FlowCreationStateProvider>
      <Consumer />
    </FlowCreationStateProvider>
  );

describe("FlowCreationContext", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    previewPlan = null;
    uiState.previewPlan = previewPlan;
    goalIds = [];
    flowCtx = null;
  });

  it("handles step change and sets derived flow id", async () => {
    renderProvider();
    const ctx = flowCtx!;

    await act(async () => {
      ctx.handleStepChange(["goal"]);
    });

    expect(uiState.setGoalSteps).toHaveBeenCalledWith(["goal"]);
    expect(uiState.updatePreviewPlan).toHaveBeenCalled();
    await act(async () => {});
    expect(flowCtx?.newID).toMatch(/goal-step-/);
  });

  it("handles empty step change and clears preview", async () => {
    renderProvider();
    const ctx = flowCtx!;

    await act(async () => {
      ctx.handleStepChange([]);
    });

    expect(uiState.clearPreviewPlan).toHaveBeenCalled();
    expect(uiState.setGoalSteps).toHaveBeenCalledWith([]);
  });

  it("creates flow successfully and reloads flows", async () => {
    renderProvider();
    let ctx = flowCtx!;
    await act(async () => {
      await ctx.handleStepChange(["goal"]);
    });
    ctx = flowCtx!;
    await act(async () => {
      ctx.setIDManuallyEdited(true);
      ctx.setNewID("flow-1");
    });
    ctx = flowCtx!;

    await act(async () => {
      ctx.handleCreateFlow();
    });

    expect(snapshotFlowPositions).toHaveBeenCalledWith("flow-1");
    expect(addFlow).toHaveBeenCalled();
    expect(loadFlows).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith("/flow/flow-1");
    expect(uiState.clearPreviewPlan).toHaveBeenCalled();
  });

  it("removes optimistic flow on create error", async () => {
    apiMock.startFlow.mockRejectedValueOnce(new Error("boom"));
    renderProvider();
    let ctx = flowCtx!;
    await act(async () => {
      await ctx.handleStepChange(["goal"]);
    });
    ctx = flowCtx!;
    await act(async () => {
      ctx.setIDManuallyEdited(true);
      ctx.setNewID("flow-err");
    });
    ctx = flowCtx!;

    await act(async () => {
      ctx.handleCreateFlow();
    });

    expect(removeFlow).toHaveBeenCalledWith("flow-err");
    expect(mockNavigate).toHaveBeenCalledWith("/");
  });
});
