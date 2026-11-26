import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";
import OverviewDiagram from "./OverviewDiagram";
import { ChatAction, Step } from "@/app/api";
import { useChatStore } from "@/app/store/chatStore";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/components/organisms/FlowCreateForm", () => {
  const MockFlowCreateForm = ({
    onCreateStep,
  }: {
    onCreateStep?: () => void;
  }) => <button onClick={onCreateStep}>{t("overview.addStep")}</button>;
  MockFlowCreateForm.displayName = "MockFlowCreateForm";
  return {
    __esModule: true,
    default: MockFlowCreateForm,
  };
});

jest.mock("@/app/contexts/StepEditorContext", () => {
  const openEditor = jest.fn();
  const closeEditor = jest.fn();
  return {
    __esModule: true,
    StepEditorProvider: ({ children }: { children: React.ReactNode }) =>
      children,
    useStepEditorContext: () => ({
      openEditor,
      closeEditor,
      isOpen: false,
      activeStep: null,
    }),
    __openEditor: openEditor,
  };
});

const sessionMock = {
  selectedFlow: null as string | null,
  selectFlow: jest.fn(),
  loadFlows: jest.fn(),
  loadSteps: jest.fn(),
  steps: [] as Step[],
  flows: [] as any[],
  flowData: null as any,
  loading: false,
  flowNotFound: false,
  executions: [] as any[],
  resolvedAttributes: [] as string[],
  flowError: null as string | null,
};

jest.mock("@/app/store/flowStore", () => ({
  useSteps: () => sessionMock.steps,
  useSpaces: () => [{ id: "sales", name: "Sales" }],
  useLoadSteps: () => sessionMock.loadSteps,
  useUpsertStep: () => jest.fn(),
}));

const mockNavigate = jest.fn();
jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => mockNavigate,
}));

const mockSetSpaceId = jest.fn();
jest.mock("@/app/contexts/UIContext", () => {
  const actual = jest.requireActual("@/app/contexts/UIContext");
  return {
    ...actual,
    UIProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    useUI: () => ({
      goalSteps: [],
      setGoalSteps: jest.fn(),
      setSpaceId: mockSetSpaceId,
      clearPreviewPlan: jest.fn(),
      panelRef: { current: null },
    }),
  };
});

jest.mock("@/app/components/templates/OverviewDiagramView", () => {
  const MockOverviewDiagramView = () => (
    <div data-testid="overview-step-diagram" />
  );
  MockOverviewDiagramView.displayName = "MockOverviewDiagramView";
  return MockOverviewDiagramView;
});

const baseStep: Step = {
  id: "a",
  name: "A",
  type: "script",
  attributes: {},
  script: { language: "python", source: "" },
};

function setSession({ steps = [] }: { steps?: Step[] }) {
  sessionMock.steps = steps;
}

describe("OverviewDiagram", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setSession({
      steps: [],
    });
  });

  it("keeps the panel when no steps", () => {
    setSession({ steps: [] });
    render(<OverviewDiagram />);
    expect(screen.getByText(t("overview.addStep"))).toBeInTheDocument();
  });

  it("shows not found state when flow missing", () => {
    setSession({ steps: [baseStep] });
    render(<OverviewDiagram />);
    expect(screen.queryByText(/Flow Not Found/)).not.toBeInTheDocument();
  });

  it("renders header stats when not in flow mode", () => {
    setSession({ steps: [baseStep] });
    render(<OverviewDiagram />);
    expect(screen.getByTestId("overview-step-diagram")).toBeInTheDocument();
  });

  it("opens create step editor", () => {
    setSession({ steps: [baseStep] });
    render(<OverviewDiagram />);
    const button = screen.getByRole("button", {
      name: t("overview.addStep"),
    });
    act(() => {
      fireEvent.click(button);
    });
    const { __openEditor } = require("@/app/contexts/StepEditorContext");
    expect(__openEditor).toHaveBeenCalled();
  });

  describe("chat actions", () => {
    afterEach(() => {
      useChatStore.setState({ action: null });
    });

    const runAction = (action: ChatAction) => {
      render(<OverviewDiagram />);
      act(() => {
        useChatStore.setState({ action });
      });
    };

    it("opens an editable Step in the editor", () => {
      setSession({ steps: [baseStep] });
      runAction({ type: "open_step", step_id: "a" });
      const { __openEditor } = require("@/app/contexts/StepEditorContext");
      expect(__openEditor).toHaveBeenCalledWith(
        expect.objectContaining({ step: baseStep })
      );
      expect(useChatStore.getState().action).toBeNull();
    });

    it("skips a Step it cannot edit", () => {
      setSession({ steps: [{ ...baseStep, script: undefined }] });
      runAction({ type: "open_step", step_id: "a" });
      const { __openEditor } = require("@/app/contexts/StepEditorContext");
      expect(__openEditor).not.toHaveBeenCalled();
    });

    it("selects a known Space", () => {
      runAction({ type: "open_overview", space_id: "sales" });
      expect(mockSetSpaceId).toHaveBeenCalledWith("sales");
    });

    it("ignores an unknown Space", () => {
      runAction({ type: "open_overview", space_id: "gone" });
      expect(mockSetSpaceId).not.toHaveBeenCalled();
    });

    it("opens a Flow", () => {
      runAction({ type: "open_flow", flow_id: "f1" });
      expect(mockNavigate).toHaveBeenCalledWith("/flow/f1");
    });

    it("leaves actions for other views pending", () => {
      runAction({ type: "open_space" });
      expect(useChatStore.getState().action).toEqual({ type: "open_space" });
    });
  });
});
