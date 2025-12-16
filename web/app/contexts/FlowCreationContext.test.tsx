import React from "react";
import { act, render } from "@testing-library/react";
import {
  FlowCreationStateProvider,
  useFlowCreation,
} from "./FlowCreationContext";
import { ExecutionPlan, Step } from "../api";

const mockRouter = {
  push: jest.fn(),
  prefetch: jest.fn(),
};

jest.mock("next/navigation", () => ({
  useRouter: () => mockRouter,
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
let showCreateForm = true;
let previewPlan: ExecutionPlan | null = null;
const uiState = {
  previewPlan,
  updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
  clearPreviewPlan: jest.fn(),
  toggleGoalStep: jest.fn(),
  get goalSteps() {
    return goalIds;
  },
  setGoalSteps: jest.fn((ids: string[]) => {
    goalIds = ids;
  }),
  get showCreateForm() {
    return showCreateForm;
  },
  setShowCreateForm: jest.fn((val: boolean) => {
    showCreateForm = val;
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
    showCreateForm = true;
    flowCtx = null;
  });

  it("handles step change and sets derived flow id", async () => {
    renderProvider();
    const ctx = flowCtx!;

    await act(async () => {
      await ctx.handleStepChange(["goal"]);
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
      await ctx.handleStepChange([]);
    });

    expect(uiState.clearPreviewPlan).toHaveBeenCalled();
    expect(uiState.setGoalSteps).toHaveBeenCalledWith([]);
  });

  it("creates flow successfully and reloads flows", async () => {
    goalIds = ["goal"];
    showCreateForm = false;
    renderProvider();
    let ctx = flowCtx!;
    await act(async () => {
      ctx.setNewID("flow-1");
    });
    ctx = flowCtx!;

    await act(async () => {
      await ctx.handleCreateFlow();
    });

    expect(addFlow).toHaveBeenCalled();
    expect(loadFlows).toHaveBeenCalled();
    expect(mockRouter.push).toHaveBeenCalledWith("/flow/flow-1");
    expect(uiState.setShowCreateForm).toHaveBeenCalledWith(false);
  });

  it("removes optimistic flow on create error", async () => {
    apiMock.startFlow.mockRejectedValueOnce(new Error("boom"));
    goalIds = ["goal"];
    showCreateForm = false;
    renderProvider();
    let ctx = flowCtx!;
    await act(async () => {
      ctx.setNewID("flow-err");
    });
    ctx = flowCtx!;

    await act(async () => {
      await ctx.handleCreateFlow();
    });

    expect(removeFlow).toHaveBeenCalledWith("flow-err");
    expect(mockRouter.push).toHaveBeenCalledWith("/");
  });
});
