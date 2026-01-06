import React from "react";
import { render, waitFor } from "@testing-library/react";
import OverviewStepDiagram from ".";
import { DiagramSelectionProvider } from "@/app/contexts/DiagramSelectionContext";
import { useEdgesState } from "@xyflow/react";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";

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
  useEdgesState: jest.fn(),
  useReactFlow: () => ({
    fitView: jest.fn(),
    setViewport: jest.fn(),
  }),
}));

jest.mock("@/utils/nodePositioning", () => ({
  loadNodePositions: jest.fn(() => ({})),
  saveNodePositions: jest.fn(),
}));

jest.mock("@/utils/stepUtils", () => ({
  sortStepsByType: jest.fn((steps) => steps),
}));

jest.mock("@/app/contexts/UIContext", () => ({
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

jest.mock("@/app/hooks/useKeyboardShortcuts", () => ({
  useKeyboardShortcuts: jest.fn(),
}));

jest.mock("./useKeyboardNavigation", () => ({
  useKeyboardNavigation: () => ({
    handleArrowUp: jest.fn(),
    handleArrowDown: jest.fn(),
    handleArrowLeft: jest.fn(),
    handleArrowRight: jest.fn(),
    handleEnter: jest.fn(),
    handleEscape: jest.fn(),
  }),
}));

jest.mock("@/app/hooks/useDiagramViewport", () => ({
  useDiagramViewport: () => ({
    handleViewportChange: jest.fn(),
    shouldFitView: true,
    savedViewport: null,
    markRestored: jest.fn(),
  }),
}));

jest.mock("./useExecutionPlanPreview", () => ({
  useExecutionPlanPreview: () => ({
    previewPlan: null,
    handleStepClick: jest.fn(),
    clearPreview: jest.fn(),
  }),
}));

jest.mock("./useStepVisibility", () => ({
  useStepVisibility: (steps: any) => ({
    visibleSteps: steps,
    previewStepIds: new Set(),
  }),
}));

jest.mock("./useNodeCalculation", () => ({
  useNodeCalculation: () => [],
}));

jest.mock("@/app/hooks/useEdgeCalculation", () => ({
  useEdgeCalculation: jest.fn(),
}));

jest.mock("./useAutoLayout", () => ({
  useAutoLayout: (nodes: any) => nodes,
}));

jest.mock("./useLayoutPlan", () => ({
  useLayoutPlan: () => ({ plan: [] }),
}));

describe("OverviewStepDiagram", () => {
  const useEdgesStateMock = useEdgesState as jest.Mock;
  const useEdgeCalculationMock = useEdgeCalculation as jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    useEdgesStateMock.mockImplementation((edges: any[]) => [
      edges,
      jest.fn(),
      jest.fn(),
    ]);
    useEdgeCalculationMock.mockReturnValue([]);
  });

  it("renders diagram scaffolding", () => {
    const { getByTestId } = render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewStepDiagram
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });

  it("renders empty state when no steps", () => {
    const { getByText } = render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewStepDiagram steps={[]} />
      </DiagramSelectionProvider>
    );

    expect(getByText("No Steps to Visualize")).toBeInTheDocument();
  });

  it("renders with goal steps selected", () => {
    const { getByTestId } = render(
      <DiagramSelectionProvider
        value={{
          goalSteps: ["s1"],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewStepDiagram
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });

  it("refreshes edges when stroke changes", async () => {
    const initialEdges = [{ id: "e1-2", style: { stroke: "#111111" } }];
    const setEdgesMock = jest.fn();
    useEdgeCalculationMock.mockReturnValue(initialEdges);
    useEdgesStateMock.mockImplementation((edges: any[]) => [
      edges,
      setEdgesMock,
      jest.fn(),
    ]);

    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewStepDiagram
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    await waitFor(() => {
      expect(setEdgesMock).toHaveBeenCalled();
    });

    const setEdgesCallback = setEdgesMock.mock.calls[0][0];
    const nextEdges = setEdgesCallback([
      { id: "e1-2", style: { stroke: "#222222" } },
    ]);

    expect(nextEdges).toBe(initialEdges);
  });
});
