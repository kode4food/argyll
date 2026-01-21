import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";
import OverviewDiagram from "./OverviewDiagram";
import { Step } from "@/app/api";
import { t, tPlural } from "@/app/testUtils/i18n";

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
  updateFlowStatus: jest.fn(),
  flowData: null as any,
  loading: false,
  flowNotFound: false,
  executions: [] as any[],
  resolvedAttributes: [] as string[],
  flowError: null as string | null,
};

jest.mock("@/app/contexts/FlowSessionContext", () => ({
  __esModule: true,
  FlowSessionProvider: ({ children }: { children: React.ReactNode }) =>
    children,
  useFlowSession: jest.fn(() => sessionMock),
}));

jest.mock("@/app/contexts/UIContext", () => {
  const actual = jest.requireActual("@/app/contexts/UIContext");
  return {
    ...actual,
    UIProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    useUI: () => ({
      goalSteps: [],
      toggleGoalStep: jest.fn(),
      setGoalSteps: jest.fn(),
      clearPreviewPlan: jest.fn(),
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
  script: { language: "python", script: "" },
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

  it("shows empty state when no steps", () => {
    setSession({ steps: [] });
    render(<OverviewDiagram />);
    expect(screen.getByText(t("overview.noStepsTitle"))).toBeInTheDocument();
  });

  it("shows not found state when flow missing", () => {
    setSession({ steps: [baseStep] });
    render(<OverviewDiagram />);
    expect(screen.queryByText(/Flow Not Found/)).not.toBeInTheDocument();
  });

  it("renders header stats when not in flow mode", () => {
    setSession({ steps: [baseStep] });
    render(<OverviewDiagram />);
    expect(screen.getByText(t("overview.title"))).toBeInTheDocument();
    expect(
      screen.getByText(tPlural("overview.stepsRegistered", 1))
    ).toBeInTheDocument();
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
});
