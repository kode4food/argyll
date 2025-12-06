import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";

import FlowDiagram from "./FlowDiagram";
import { Step, FlowContext, ExecutionResult } from "../../api";

jest.mock("../../hooks/useFlowWebSocket", () => ({
  useFlowWebSocket: jest.fn(),
}));

jest.mock("../../store/flowStore", () => {
  const actual = jest.requireActual("../../store/flowStore");
  return {
    ...actual,
    useSteps: jest.fn(),
    useSelectedFlow: jest.fn(),
    useFlowData: jest.fn(),
    useExecutions: jest.fn(),
    useResolvedAttributes: jest.fn(),
    useFlowLoading: jest.fn(),
    useFlowNotFound: jest.fn(),
    useIsFlowMode: jest.fn(),
    useLoadSteps: jest.fn(),
  };
});

jest.mock("../../contexts/UIContext", () => {
  const actual = jest.requireActual("../../contexts/UIContext");
  return {
    ...actual,
    UIProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    useUI: () => ({
      selectedStep: null,
      setSelectedStep: jest.fn(),
    }),
  };
});

jest.mock("./StepDiagram", () => {
  const MockStepDiagram = () => <div data-testid="step-diagram" />;
  MockStepDiagram.displayName = "MockStepDiagram";
  return MockStepDiagram;
});
jest.mock("../organisms/StepEditor", () => {
  const MockStepEditor = (props: any) => (
    <div data-testid="step-editor">Step Editor</div>
  );
  MockStepEditor.displayName = "MockStepEditor";
  return MockStepEditor;
});

const flowStore = require("../../store/flowStore") as jest.Mocked<
  typeof import("../../store/flowStore")
>;

const baseStep: Step = {
  id: "a",
  name: "A",
  type: "script",
  attributes: {},
  version: "1",
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

function setStore({
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
  flowStore.useSteps.mockReturnValue(steps);
  flowStore.useSelectedFlow.mockReturnValue(selectedFlow);
  flowStore.useFlowData.mockReturnValue(flowData);
  flowStore.useExecutions.mockReturnValue(executions);
  flowStore.useResolvedAttributes.mockReturnValue(resolved);
  flowStore.useFlowLoading.mockReturnValue(loading);
  flowStore.useFlowNotFound.mockReturnValue(flowNotFound);
  flowStore.useIsFlowMode.mockReturnValue(isFlowMode);
  flowStore.useLoadSteps.mockReturnValue(jest.fn());
}

describe("FlowDiagram", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("shows empty state when no steps", () => {
    setStore({ steps: [] });
    render(<FlowDiagram />);
    expect(screen.getByText("No Steps Registered")).toBeInTheDocument();
  });

  it("shows not found state when flow missing", () => {
    setStore({
      steps: [baseStep],
      selectedFlow: "wf-1",
      flowNotFound: true,
    });
    render(<FlowDiagram />);
    expect(screen.getByText(/Flow Not Found/)).toBeInTheDocument();
  });

  it("renders header stats when not in flow mode", () => {
    setStore({
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
    setStore({
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
    setStore({
      steps: [baseStep],
      isFlowMode: false,
      flowData: null,
    });
    render(<FlowDiagram />);
    const button = screen.getByRole("button", { name: /Create New Step/i });
    act(() => {
      fireEvent.click(button);
    });
    expect(screen.getByTestId("step-editor")).toBeInTheDocument();
  });
});
