import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";

import FlowDiagram from "./FlowDiagram";

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

jest.mock("./StepDiagram", () => () => <div data-testid="step-diagram" />);
jest.mock("../organisms/StepEditor", () => (props: any) => (
  <div data-testid="step-editor">Step Editor</div>
));

const flowStore = require("../../store/flowStore");

function setStore({
  steps = [],
  selectedFlow = null,
  flowData = null,
  executions = null,
  resolved = [],
  loading = false,
  flowNotFound = false,
  isFlowMode = false,
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
      steps: [{ id: "a", name: "A" }],
      selectedFlow: "wf-1",
      flowNotFound: true,
    });
    render(<FlowDiagram />);
    expect(screen.getByText(/Flow Not Found/)).toBeInTheDocument();
  });

  it("renders header stats when not in flow mode", () => {
    setStore({
      steps: [{ id: "a", name: "A" }],
      flowData: {
        id: "wf-1",
        status: "active",
        started_at: Date.now(),
        plan: { steps: { a: {} } },
      },
      resolved: [],
      isFlowMode: false,
    });
    render(<FlowDiagram />);
    expect(screen.getByText("Step Dependencies")).toBeInTheDocument();
    expect(screen.getByText(/1 step registered/)).toBeInTheDocument();
  });

  it("renders flow header when in flow mode", () => {
    setStore({
      steps: [{ id: "a", name: "A" }],
      flowData: {
        id: "wf-1",
        status: "active",
        started_at: Date.now(),
        completed_at: Date.now(),
        plan: { steps: { a: {} } },
      },
      resolved: [],
      isFlowMode: true,
      executions: [],
    });
    render(<FlowDiagram />);
    expect(screen.getByText("wf-1")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("opens create step editor", () => {
    setStore({
      steps: [{ id: "a", name: "A" }],
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
