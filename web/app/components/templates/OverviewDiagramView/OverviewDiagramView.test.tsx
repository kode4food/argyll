import React from "react";
import { render, screen } from "@testing-library/react";
import OverviewDiagramView from ".";
import { t } from "@/app/testUtils/i18n";
import { DiagramSelectionProvider } from "@/app/contexts/DiagramSelectionContext";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";

const reactFlowMock = jest.fn(() => <div data-testid="react-flow" />);
const previewHookState = {
  previewPlan: null as any,
};

jest.mock("@xyflow/react", () => ({
  ReactFlow: (props: any) => reactFlowMock(props),
  Controls: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="controls">{children}</div>
  ),
  ControlButton: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  PanelPosition: {
    BottomRight: "bottom-right",
  },
  MiniMap: () => <div data-testid="mini-map" />,
  Background: () => <div data-testid="background" />,
  BackgroundVariant: { Dots: "dots" },
  ReactFlowProvider: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  useNodesState: (nodes: any[]) => [nodes, jest.fn(), jest.fn()],
  useReactFlow: () => ({
    fitView: jest.fn(),
    setViewport: jest.fn(),
    zoomIn: jest.fn(),
    zoomOut: jest.fn(),
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
    focusedPreviewAttribute: null,
    setFocusedPreviewAttribute: jest.fn(),
    setPreviewPlan: jest.fn(),
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
    previewPlan: null,
    diagramContainerRef: { current: null },
    headerRef: { current: null },
    panelRef: { current: null },
  }),
}));

jest.mock("@/app/hooks/useFitView", () => ({
  useFitView: () => jest.fn(),
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
    markFitApplied: jest.fn(),
  }),
}));

jest.mock("./useExecutionPlanPreview", () => ({
  useExecutionPlanPreview: () => ({
    previewPlan: previewHookState.previewPlan,
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

describe("OverviewDiagramView", () => {
  const useEdgeCalculationMock = useEdgeCalculation as jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    useEdgeCalculationMock.mockReturnValue([]);
    previewHookState.previewPlan = null;
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
        <OverviewDiagramView
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
        <OverviewDiagramView steps={[]} />
      </DiagramSelectionProvider>
    );

    expect(getByText(t("overview.noVisibleTitle"))).toBeInTheDocument();
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
        <OverviewDiagramView
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });

  it("renders floating preview hud when preview is active", () => {
    previewHookState.previewPlan = {
      goals: ["s1"],
      steps: {
        s1: {
          step: { id: "s1", name: "Step 1", type: "sync", attributes: {} },
          inputs: {},
        },
        s2: {
          step: { id: "s2", name: "Step 2", type: "sync", attributes: {} },
          inputs: {},
        },
      },
      excluded: [],
    };

    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: ["s1"],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    expect(screen.getByText(t("overview.previewLabel"))).toBeInTheDocument();
    expect(
      screen.getByText(t("overview.previewGoals", { count: 1 }))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("overview.previewSteps", { count: 2 }))
    ).toBeInTheDocument();
  });

  it("passes calculated edges directly to ReactFlow", () => {
    const initialEdges = [{ id: "e1-2", style: { stroke: "#111111" } }];
    useEdgeCalculationMock.mockReturnValue(initialEdges);

    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          toggleGoalStep: jest.fn(),
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[{ id: "s1", name: "Step 1", type: "sync", attributes: {} }]}
        />
      </DiagramSelectionProvider>
    );

    expect(reactFlowMock).toHaveBeenCalled();
    const props = reactFlowMock.mock.calls[0][0];
    expect(props.edges).toBe(initialEdges);
  });
});
