import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";

import FlowSelector from "./FlowSelector";
import { useFlowCreation } from "../../contexts/FlowCreationContext";

const pushMock = jest.fn();
const subscribeMock = jest.fn();
let eventsMock: any[] = [];

const uiState: any = {
  showCreateForm: false,
  setShowCreateForm: jest.fn(),
  previewPlan: null,
  updatePreviewPlan: jest.fn(),
  clearPreviewPlan: jest.fn(),
  setSelectedStep: jest.fn(),
  goalStepIds: [],
  setGoalStepIds: jest.fn(),
};

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
    prefetch: jest.fn(),
  }),
  useParams: () => ({}),
  usePathname: () => "/",
}));

jest.mock("../../hooks/useFlowFromUrl", () => ({
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

jest.mock("../../store/flowStore", () => {
  const actual = jest.requireActual("../../store/flowStore");
  const useFlows = jest.fn(() => []);
  const useSteps = jest.fn(() => []);
  const useLoadFlows = jest.fn(() => jest.fn());
  const useAddFlow = jest.fn(() => jest.fn());
  const useRemoveFlow = jest.fn(() => jest.fn());
  const useUpdateFlowStatus = jest.fn(() => jest.fn());

  return {
    ...actual,
    useFlows,
    useSelectedFlow: jest.fn(() => null),
    useSteps,
    useLoadFlows,
    useAddFlow,
    useRemoveFlow,
    useUpdateFlowStatus,
  };
});

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

let capturedFormProps: any = null;
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
  beforeEach(() => {
    jest.clearAllMocks();
    eventsMock = [];
    capturedFormProps = null;
    Object.assign(uiState, {
      showCreateForm: false,
      previewPlan: null,
      goalStepIds: [],
      updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
      setGoalStepIds: jest.fn(),
      setShowCreateForm: jest.fn(),
      setSelectedStep: jest.fn(),
      clearPreviewPlan: jest.fn(),
    });
  });

  it("renders and can open dropdown", () => {
    render(<FlowSelector />);
    const button = screen.getByRole("button", { name: /Select Flow/i });
    fireEvent.click(button);
    expect(screen.getByPlaceholderText(/Search flows/)).toBeInTheDocument();
  });

  it("shows new flow button when no selection", () => {
    render(<FlowSelector />);
    expect(
      screen.getByRole("button", { name: /Create New Flow/i })
    ).toBeInTheDocument();
  });

  it("pushes route when selecting a flow from dropdown", () => {
    require("../../store/flowStore").useFlows.mockReturnValue([
      { id: "wf-1", status: "pending" },
      { id: "wf-2", status: "completed" },
    ]);

    render(<FlowSelector />);
    fireEvent.click(screen.getByRole("button", { name: /Select Flow/i }));
    fireEvent.mouseDown(screen.getByText("wf-1"));

    expect(pushMock).toHaveBeenCalledWith("/flow/wf-1");
  });

  it("subscribes on mount and updates flow status from events", () => {
    const updateFlowStatus = jest.fn();
    require("../../store/flowStore").useUpdateFlowStatus.mockReturnValue(
      updateFlowStatus
    );
    eventsMock = [
      {
        type: "flow_completed",
        timestamp: Date.now(),
        sequence: 1,
        id: ["flow", "wf-123"],
      },
    ];

    render(<FlowSelector />);

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
    uiState.goalStepIds = [];
    const { useSteps } = require("../../store/flowStore");
    useSteps.mockReturnValue([{ id: "goal", name: "Goal" }]);

    render(<FlowSelector />);
    await screen.findByText("FlowCreateForm");
    expect(capturedFormProps).not.toBeNull();

    await act(async () => {
      await capturedFormProps.handleStepChange(["goal"]);
    });

    expect(uiState.setGoalStepIds).toHaveBeenCalledWith(["goal"]);
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
    uiState.goalStepIds = ["goal"];
    const {
      useSteps,
      useAddFlow,
      useRemoveFlow,
    } = require("../../store/flowStore");
    useSteps.mockReturnValue([{ id: "goal", name: "Goal" }]);

    const addFlow = jest.fn();
    const removeFlow = jest.fn();
    useAddFlow.mockReturnValue(addFlow);
    useRemoveFlow.mockReturnValue(removeFlow);

    render(<FlowSelector />);
    await screen.findByText("FlowCreateForm");
    expect(capturedFormProps).not.toBeNull();

    await act(async () => {
      capturedFormProps.setNewID("new-flow");
    });
    await act(async () => {
      await capturedFormProps.handleStepChange(["goal"]);
    });
    await act(async () => {
      await capturedFormProps.handleCreateFlow();
    });

    expect(addFlow).toHaveBeenCalled();
    expect(removeFlow).toHaveBeenCalled();
  });
});
