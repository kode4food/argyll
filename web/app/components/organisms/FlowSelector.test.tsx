import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";

import FlowSelector from "./FlowSelector";
import { useFlowCreation } from "../../contexts/FlowCreationContext";
import { FlowSessionProvider } from "../../contexts/FlowSessionContext";
import { FlowContext, Step } from "../../api";
import { WebSocketEvent } from "../../hooks/useWebSocketContext";

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
const subscribeMock = jest.fn();
let eventsMock: WebSocketEvent[] = [];

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

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
    prefetch: jest.fn(),
  }),
  useParams: () => ({}),
  usePathname: () => "/",
}));

jest.mock("./FlowSelector/useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("../../hooks/useWebSocketContext", () => ({
  useWebSocketContext: () => ({
    subscribe: subscribeMock,
    events: eventsMock,
  }),
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
    updateFlowStatus: jest.fn(),
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
    eventsMock = [];
    capturedFormProps = null;
    flowCreationMock.newID = "";
    flowCreationMock.handleCreateFlow = jest.fn();
    flowCreationMock.handleStepChange = jest.fn(async (ids: string[]) => {
      uiState.setGoalSteps(ids);
      await uiState.updatePreviewPlan(ids, {});
    });
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

  it("pushes route when selecting a flow from dropdown", async () => {
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

  it("subscribes on mount and updates flow status from events", async () => {
    const updateFlowStatus = jest.fn();
    flowSessionMock.updateFlowStatus = updateFlowStatus;
    eventsMock = [
      {
        type: "flow_completed",
        timestamp: Date.now(),
        sequence: 1,
        id: ["flow", "wf-123"],
        data: {},
      },
    ];

    await renderSelector();

    expect(subscribeMock).toHaveBeenCalledWith({
      event_types: ["flow_started", "flow_completed", "flow_failed"],
    });
    expect(updateFlowStatus).toHaveBeenCalledWith(
      "wf-123",
      "completed",
      expect.any(String)
    );
  });

  it("handles goal step change and updates preview plan", async () => {
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
      await capturedFormProps!.handleStepChange(["goal"]);
    });

    expect(uiState.setGoalSteps).toHaveBeenCalledWith(["goal"]);
    expect(uiState.updatePreviewPlan).toHaveBeenCalledWith(
      ["goal"],
      expect.any(Object)
    );
  });

  it("handles create flow error and removes optimistic flow", async () => {
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
      await capturedFormProps!.handleStepChange(["goal"]);
    });
    await act(async () => {
      await capturedFormProps!.handleCreateFlow();
    });

    expect(addFlow).toHaveBeenCalled();
    expect(removeFlow).toHaveBeenCalled();
  });
});
