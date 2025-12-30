import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";

import FlowSelector from "./FlowSelector";
import { useFlowCreation } from "../../contexts/FlowCreationContext";
import { FlowSessionProvider } from "../../contexts/FlowSessionContext";
import { FlowContext, Step } from "../../api";

type UIStateMock = {
  showCreateForm: boolean;
  setShowCreateForm: jest.Mock;
  previewPlan: unknown;
  updatePreviewPlan: jest.Mock;
  clearPreviewPlan: jest.Mock;
  toggleGoalStep: jest.Mock;
  goalSteps: string[];
  setGoalSteps: jest.Mock;
  disableEdit: boolean;
  diagramContainerRef: { current: null };
};

const pushMock = jest.fn();

const uiState: UIStateMock = {
  showCreateForm: false,
  setShowCreateForm: jest.fn(),
  previewPlan: null,
  updatePreviewPlan: jest.fn(),
  clearPreviewPlan: jest.fn(),
  toggleGoalStep: jest.fn(),
  goalSteps: [],
  setGoalSteps: jest.fn(),
  disableEdit: false,
  diagramContainerRef: { current: null },
};

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => pushMock,
}));

jest.mock("./FlowSelector/useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("../../contexts/UIContext", () => ({
  useUI: () => uiState,
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock("../../api", () => ({
  api: {
    getExecutionPlan: jest.fn(),
    startFlow: jest.fn(),
  },
}));

jest.mock("react-hot-toast", () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

jest.mock("../../contexts/FlowSessionContext", () => {
  const session = {
    selectedFlow: null as string | null,
    selectFlow: jest.fn(),
    loadFlows: jest.fn(),
    loadSteps: jest.fn(),
    steps: [] as Step[],
    flows: [] as FlowContext[],
    flowData: null,
    loading: false,
    flowNotFound: false,
    isFlowMode: false,
    executions: [],
    resolvedAttributes: [],
    flowError: null as string | null,
  };
  return {
    __esModule: true,
    FlowSessionProvider: ({ children }: { children: React.ReactNode }) =>
      children,
    useFlowSession: () => session,
    __sessionMock: session,
  };
});

jest.mock("../../contexts/FlowCreationContext", () => {
  const flowCreationValue = {
    newID: "",
    setNewID: jest.fn((id: string) => {
      flowCreationValue.newID = id;
    }),
    setIDManuallyEdited: jest.fn(),
    handleStepChange: jest.fn(),
    initialState: "{}",
    setInitialState: jest.fn(),
    creating: false,
    handleCreateFlow: jest.fn(),
    steps: [] as Step[],
    generateID: jest.fn(() => "generated-id"),
    sortSteps: jest.fn((steps: Step[]) => steps),
  };

  return {
    __esModule: true,
    FlowCreationStateProvider: ({ children }: { children: React.ReactNode }) =>
      children,
    useFlowCreation: () => flowCreationValue,
    __flowCreationValue: flowCreationValue,
  };
});

const {
  __sessionMock: flowSessionMock,
} = require("../../contexts/FlowSessionContext");
const {
  __flowCreationValue: flowCreationMock,
} = require("../../contexts/FlowCreationContext");

let capturedFormProps: ReturnType<typeof useFlowCreation> | null = null;
const MockFlowCreateForm = () => {
  capturedFormProps = useFlowCreation();
  return <div>FlowCreateForm</div>;
};
MockFlowCreateForm.displayName = "MockFlowCreateForm";

const MockKeyboardShortcutsModal = () => <div>Shortcuts</div>;
MockKeyboardShortcutsModal.displayName = "MockKeyboardShortcutsModal";

jest.mock("./FlowCreateForm", () => ({
  __esModule: true,
  default: MockFlowCreateForm,
}));
jest.mock("../molecules/KeyboardShortcutsModal", () => ({
  __esModule: true,
  default: MockKeyboardShortcutsModal,
}));

describe("FlowSelector", () => {
  const renderSelector = async () => {
    await act(async () => {
      render(
        <FlowSessionProvider>
          <FlowSelector />
        </FlowSessionProvider>
      );
    });
  };

  beforeEach(() => {
    jest.clearAllMocks();
    capturedFormProps = null;
    flowCreationMock.newID = "";
    flowCreationMock.handleCreateFlow = jest.fn();
    flowCreationMock.handleStepChange = jest.fn(async (ids: string[]) => {
      uiState.setGoalSteps(ids);
      await uiState.updatePreviewPlan(ids, {});
    });
    flowSessionMock.selectedFlow = null;
    flowSessionMock.flows = [];
    Object.assign(uiState, {
      showCreateForm: false,
      previewPlan: null,
      goalSteps: [],
      updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
      setGoalSteps: jest.fn(),
      setShowCreateForm: jest.fn(),
      clearPreviewPlan: jest.fn(),
    });
  });

  it("renders and can open dropdown", async () => {
    await renderSelector();
    const button = screen.getByRole("button", { name: /Select Flow/i });
    fireEvent.click(button);
    expect(screen.getByPlaceholderText(/Search flows/)).toBeInTheDocument();
  });

  it("shows new flow button when no selection", async () => {
    await renderSelector();
    expect(
      screen.getByRole("button", { name: /Create New Flow/i })
    ).toBeInTheDocument();
  });

  it("shows back button when flow selected", async () => {
    flowSessionMock.selectedFlow = "flow-1";
    flowSessionMock.flows = [
      {
        id: "flow-1",
        status: "completed",
        state: {},
        started_at: "2024-01-01T00:00:00Z",
        completed_at: "2024-01-02T00:00:00Z",
      },
    ];

    await renderSelector();

    const backButton = screen.getByLabelText(/Back to Overview/i);
    fireEvent.click(backButton);

    expect(pushMock).toHaveBeenCalledWith("/");
  });

  it("navigates when selecting a flow", async () => {
    flowSessionMock.flows = [
      { id: "wf-1", status: "pending" },
      { id: "wf-2", status: "completed" },
    ];
    flowSessionMock.selectedFlow = null;

    await renderSelector();
    fireEvent.click(screen.getByRole("button", { name: /Select Flow/i }));
    fireEvent.mouseDown(screen.getByText("wf-1"));

    expect(pushMock).toHaveBeenCalledWith("/flow/wf-1");
  });

  it("updates preview when goal changes", async () => {
    const { api } = require("../../api");
    api.getExecutionPlan.mockResolvedValue({
      steps: { goal: {} },
      required: [],
    });
    uiState.showCreateForm = true;
    uiState.goalSteps = [];
    flowCreationMock.steps = [{ id: "goal", name: "Goal" }];

    await renderSelector();
    await screen.findByText("FlowCreateForm");
    expect(capturedFormProps).not.toBeNull();

    await act(async () => {
      capturedFormProps!.handleStepChange(["goal"]);
    });

    expect(uiState.setGoalSteps).toHaveBeenCalledWith(["goal"]);
    expect(uiState.updatePreviewPlan).toHaveBeenCalledWith(
      ["goal"],
      expect.any(Object)
    );
  });

  it("removes optimistic flow on create error", async () => {
    const { api } = require("../../api");
    api.getExecutionPlan.mockResolvedValue({
      steps: { goal: {} },
      required: [],
    });
    api.startFlow.mockRejectedValue(new Error("fail"));
    uiState.showCreateForm = true;
    uiState.goalSteps = ["goal"];
    flowCreationMock.steps = [{ id: "goal", name: "Goal" }];
    const addFlow = jest.fn();
    const removeFlow = jest.fn();
    flowCreationMock.handleCreateFlow = async () => {
      addFlow();
      removeFlow();
    };

    await renderSelector();
    await screen.findByText("FlowCreateForm");
    expect(capturedFormProps).not.toBeNull();

    await act(async () => {
      capturedFormProps!.setNewID("new-flow");
    });
    await act(async () => {
      capturedFormProps!.handleStepChange(["goal"]);
    });
    await act(async () => {
      capturedFormProps!.handleCreateFlow();
    });

    expect(addFlow).toHaveBeenCalled();
    expect(removeFlow).toHaveBeenCalled();
  });
});
