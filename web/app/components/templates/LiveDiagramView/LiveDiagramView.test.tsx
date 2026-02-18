import React from "react";
import { render, screen } from "@testing-library/react";
import LiveDiagramView from "./LiveDiagramView";
import { Step, FlowContext, ExecutionResult } from "@/app/api";

jest.mock("@xyflow/react", () => ({
  ReactFlow: () => <div data-testid="react-flow" />,
  Controls: () => <div data-testid="controls" />,
  Background: () => <div data-testid="background" />,
  BackgroundVariant: { Dots: "dots" },
  ReactFlowProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useReactFlow: () => ({
    fitView: jest.fn(),
    setViewport: jest.fn(),
  }),
}));

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({
    disableEdit: false,
    diagramContainerRef: { current: null },
  }),
}));

jest.mock("@/app/hooks/useDiagramViewport", () => ({
  useDiagramViewport: () => ({
    handleViewportChange: jest.fn(),
    shouldFitView: true,
    savedViewport: null,
    markRestored: jest.fn(),
    markFitApplied: jest.fn(),
  }),
}));

const mockUseStepVisibility = jest.fn();
const mockUseNodeCalculation = jest.fn();
const mockUseEdgeCalculation = jest.fn();

jest.mock("./useStepVisibility", () => ({
  useStepVisibility: (...args: any[]) => mockUseStepVisibility(...args),
}));

jest.mock("./useNodeCalculation", () => ({
  useNodeCalculation: (...args: any[]) => mockUseNodeCalculation(...args),
}));

jest.mock("@/app/hooks/useEdgeCalculation", () => ({
  useEdgeCalculation: (...args: any[]) => mockUseEdgeCalculation(...args),
}));

const baseStep: Step = {
  id: "a",
  name: "Step A",
  type: "sync",
  attributes: {},
};

const makeFlowData = (overrides?: Partial<FlowContext>): FlowContext => ({
  id: "wf-1",
  status: "active",
  state: {},
  plan: {
    goals: ["a"],
    required: [],
    steps: {
      a: baseStep,
    },
    attributes: {},
  },
  started_at: new Date().toISOString(),
  ...overrides,
});

describe("LiveDiagramView", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseStepVisibility.mockReturnValue({
      visibleSteps: [baseStep],
    });
    mockUseNodeCalculation.mockReturnValue([]);
    mockUseEdgeCalculation.mockReturnValue([]);
  });

  test("renders empty state when no visible steps", () => {
    mockUseStepVisibility.mockReturnValue({
      visibleSteps: [],
    });

    render(
      <LiveDiagramView
        steps={[]}
        flowData={makeFlowData()}
        executions={[]}
        resolvedAttributes={[]}
      />
    );

    expect(screen.queryByTestId("react-flow")).not.toBeInTheDocument();
  });

  test("renders ReactFlow when visible steps exist", () => {
    render(
      <LiveDiagramView
        steps={[baseStep]}
        flowData={makeFlowData()}
        executions={[]}
        resolvedAttributes={[]}
      />
    );

    expect(screen.getByTestId("react-flow")).toBeInTheDocument();
  });

  test("passes correct props to useNodeCalculation", () => {
    const flowData = makeFlowData();
    const executions: ExecutionResult[] = [
      {
        step_id: "a",
        flow_id: "wf-1",
        status: "succeeded",
        inputs: {},
        started_at: new Date().toISOString(),
      },
    ];
    const resolvedAttributes = ["attr1"];

    render(
      <LiveDiagramView
        steps={[baseStep]}
        flowData={flowData}
        executions={executions}
        resolvedAttributes={resolvedAttributes}
      />
    );

    const calls = mockUseNodeCalculation.mock.calls;
    expect(calls.length).toBeGreaterThan(0);
    expect(calls[0][0]).toEqual([baseStep]);
    expect(calls[0][1]).toBe(flowData);
    expect(calls[0][2]).toBe(executions);
    expect(calls[0][3]).toBe(resolvedAttributes);
    expect(calls[0][4]).toHaveProperty("current");
    expect(calls[0][5]).toBe(false);
  });

  test("passes correct props to useEdgeCalculation", () => {
    render(
      <LiveDiagramView
        steps={[baseStep]}
        flowData={makeFlowData()}
        executions={[]}
        resolvedAttributes={[]}
      />
    );

    expect(mockUseEdgeCalculation).toHaveBeenCalledWith([baseStep], null);
  });

  test("handles default props", () => {
    mockUseStepVisibility.mockReturnValue({
      visibleSteps: [],
    });

    render(<LiveDiagramView steps={[]} flowData={null} />);

    expect(screen.queryByTestId("react-flow")).not.toBeInTheDocument();
  });

  test("renders empty state when flow has no plan", () => {
    render(
      <LiveDiagramView
        steps={[baseStep]}
        flowData={makeFlowData({ plan: undefined })}
        executions={[]}
        resolvedAttributes={[]}
      />
    );

    expect(screen.queryByTestId("react-flow")).not.toBeInTheDocument();
  });
});
