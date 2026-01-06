import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";
import FlowDiagram from "./FlowDiagram";
import { Step, FlowContext, ExecutionResult } from "@/app/api";

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
  flows: [] as FlowContext[],
  updateFlowStatus: jest.fn(),
  flowData: null as FlowContext | null,
  loading: false,
  flowNotFound: false,
  isFlowMode: false,
  executions: [] as ExecutionResult[],
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

jest.mock("@/app/components/templates/OverviewStepDiagram", () => {
  const MockOverviewStepDiagram = () => (
    <div data-testid="overview-step-diagram" />
  );
  MockOverviewStepDiagram.displayName = "MockOverviewStepDiagram";
  return MockOverviewStepDiagram;
});

jest.mock("@/app/components/templates/LiveFlowStepDiagram", () => {
  const MockLiveFlowStepDiagram = () => (
    <div data-testid="live-flow-step-diagram" />
  );
  MockLiveFlowStepDiagram.displayName = "MockLiveFlowStepDiagram";
  return MockLiveFlowStepDiagram;
});

const baseStep: Step = {
  id: "a",
  name: "A",
  type: "script",
  attributes: {},
  script: { language: "python", script: "" },
};

const basePlan = {
  goals: [baseStep.id],
  required: [],
  steps: { [baseStep.id]: baseStep },
  attributes: {},
};

const makeFlow = (overrides: Partial<FlowContext>): FlowContext => ({
  id: "wf-1",
  status: "active",
  state: {},
  started_at: new Date().toISOString(),
  ...overrides,
});

const makeExecutions = (
  list: Partial<ExecutionResult>[] = []
): ExecutionResult[] =>
  list.map((exec) => ({
    step_id: "a",
    flow_id: "wf-1",
    status: "pending",
    inputs: {},
    started_at: new Date().toISOString(),
    ...exec,
  }));

function setSession({
  steps = [],
  selectedFlow = null,
  flowData = null,
  executions = [],
  resolved = [],
  loading = false,
  flowNotFound = false,
  isFlowMode = false,
}: {
  steps?: Step[];
  selectedFlow?: string | null;
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolved?: string[];
  loading?: boolean;
  flowNotFound?: boolean;
  isFlowMode?: boolean;
}) {
  sessionMock.steps = steps;
  sessionMock.selectedFlow = selectedFlow;
  sessionMock.flowData = flowData;
  sessionMock.executions = executions;
  sessionMock.resolvedAttributes = resolved;
  sessionMock.loading = loading;
  sessionMock.flowNotFound = flowNotFound;
  sessionMock.isFlowMode = isFlowMode;
}

describe("FlowDiagram", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setSession({
      steps: [],
      selectedFlow: null,
      flowData: null,
      executions: [],
      resolved: [],
      loading: false,
      flowNotFound: false,
      isFlowMode: false,
    });
  });

  it("shows empty state when no steps", () => {
    setSession({ steps: [] });
    render(<FlowDiagram />);
    expect(screen.getByText("No Steps Registered")).toBeInTheDocument();
  });

  it("shows not found state when flow missing", () => {
    setSession({
      steps: [baseStep],
      selectedFlow: "wf-1",
      flowNotFound: true,
    });
    render(<FlowDiagram />);
    expect(screen.getByText(/Flow Not Found/)).toBeInTheDocument();
  });

  it("renders header stats when not in flow mode", () => {
    setSession({
      steps: [baseStep],
      flowData: makeFlow({ plan: basePlan }),
      resolved: [],
      isFlowMode: false,
    });
    render(<FlowDiagram />);
    expect(screen.getByText("Step Dependencies")).toBeInTheDocument();
    expect(screen.getByText(/1 step registered/)).toBeInTheDocument();
  });

  it("renders flow header when in flow mode", () => {
    setSession({
      steps: [baseStep],
      flowData: makeFlow({
        completed_at: new Date().toISOString(),
        plan: basePlan,
      }),
      resolved: [],
      isFlowMode: true,
      executions: makeExecutions([]),
    });
    render(<FlowDiagram />);
    expect(screen.getByText("wf-1")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("opens create step editor", () => {
    setSession({
      steps: [baseStep],
      isFlowMode: false,
      flowData: null,
    });
    render(<FlowDiagram />);
    const button = screen.getByRole("button", { name: /Create New Step/i });
    act(() => {
      fireEvent.click(button);
    });
    const { __openEditor } = require("@/app/contexts/StepEditorContext");
    expect(__openEditor).toHaveBeenCalled();
  });
});
