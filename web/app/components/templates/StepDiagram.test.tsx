import { render } from "@testing-library/react";

import StepDiagram from "./StepDiagram";
import { DiagramSelectionProvider } from "../../contexts/DiagramSelectionContext";

jest.mock("@xyflow/react", () => ({
  ReactFlow: () => <div data-testid="react-flow" />,
  MiniMap: () => <div data-testid="mini-map" />,
  Controls: () => <div data-testid="controls" />,
  Background: () => <div data-testid="background" />,
  BackgroundVariant: { Dots: "dots" },
  ReactFlowProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useNodesState: (nodes: any[]) => [nodes, jest.fn(), jest.fn()],
  useEdgesState: (edges: any[]) => [edges, jest.fn(), jest.fn()],
  useReactFlow: () => ({
    fitView: jest.fn(),
    setViewport: jest.fn(),
  }),
}));

jest.mock("./StepDiagram/nodePositioning", () => ({
  loadNodePositions: jest.fn(() => ({})),
  saveNodePositions: jest.fn(),
}));

jest.mock("@/utils/stepUtils", () => ({
  sortStepsByType: jest.fn((steps) => steps),
}));

jest.mock("../../contexts/UIContext", () => ({
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useUI: () => ({
    goalSteps: [],
    toggleGoalStep: jest.fn(),
    setGoalSteps: jest.fn(),
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
    previewPlan: null,
    disableEdit: false,
    diagramContainerRef: { current: null },
  }),
}));

describe("StepDiagram", () => {
  it("renders diagram scaffolding", () => {
    const { getByTestId } = render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <StepDiagram
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
          flowData={null}
          executions={[]}
          resolvedAttributes={[]}
        />
      </DiagramSelectionProvider>
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });
});
