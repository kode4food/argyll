import React from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import FlowSelector from "./FlowSelector";
import { t } from "@/app/testUtils/i18n";
import { FlowSessionProvider } from "@/app/contexts/FlowSessionContext";
import { FlowContext, Step } from "@/app/api";

type UIStateMock = {
  previewPlan: unknown;
  setPreviewPlan: jest.Mock;
  updatePreviewPlan: jest.Mock;
  clearPreviewPlan: jest.Mock;
  toggleGoalStep: jest.Mock;
  goalSteps: string[];
  setGoalSteps: jest.Mock;
  diagramContainerRef: { current: null };
};

const pushMock = jest.fn();

const uiState: UIStateMock = {
  previewPlan: null,
  setPreviewPlan: jest.fn(),
  updatePreviewPlan: jest.fn(),
  clearPreviewPlan: jest.fn(),
  toggleGoalStep: jest.fn(),
  goalSteps: [],
  setGoalSteps: jest.fn(),
  diagramContainerRef: { current: null },
};

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => pushMock,
}));

jest.mock("./useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => uiState,
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock("@/app/api", () => ({
  api: {
    getExecutionPlan: jest.fn(),
    startFlow: jest.fn(),
  },
}));

jest.mock("react-hot-toast", () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

jest.mock("@/app/contexts/FlowSessionContext", () => {
  const session = {
    selectedFlow: null as string | null,
    selectFlow: jest.fn(),
    loadFlows: jest.fn(),
    loadSteps: jest.fn(),
    steps: [] as Step[],
    flows: [] as FlowContext[],
    flowsHasMore: false,
    flowsLoading: false,
    loadMoreFlows: jest.fn(),
    flowData: null,
    loading: false,
    flowNotFound: false,
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

const {
  __sessionMock: flowSessionMock,
} = require("@/app/contexts/FlowSessionContext");

const MockKeyboardShortcutsModal = () => <div>Shortcuts</div>;
MockKeyboardShortcutsModal.displayName = "MockKeyboardShortcutsModal";
jest.mock("@/app/components/molecules/KeyboardShortcutsModal", () => ({
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
    flowSessionMock.selectedFlow = null;
    flowSessionMock.flows = [];
    Object.assign(uiState, {
      previewPlan: null,
      setPreviewPlan: jest.fn(),
      goalSteps: [],
      updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
      setGoalSteps: jest.fn(),
      clearPreviewPlan: jest.fn(),
    });
  });

  it("renders and can open dropdown", async () => {
    await renderSelector();
    const button = screen.getByRole("button", {
      name: t("flowSelector.selectFlow"),
    });
    fireEvent.click(button);
    expect(
      screen.getByPlaceholderText(t("flowSelector.searchPlaceholder"))
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

    const backButton = screen.getByLabelText(t("flowSelector.backToOverview"));
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
    fireEvent.click(
      screen.getByRole("button", { name: t("flowSelector.selectFlow") })
    );
    fireEvent.mouseDown(screen.getByText("wf-1"));

    expect(pushMock).toHaveBeenCalledWith("/flow/wf-1");
  });
});
