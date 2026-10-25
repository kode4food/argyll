import React from "react";
import { render, screen } from "@testing-library/react";
import OverviewDiagramView from "./OverviewDiagramView";
import { t } from "@/app/testUtils/i18n";
import { DiagramSelectionProvider } from "@/app/contexts/DiagramSelectionContext";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { AttributeRole, AttributeType, Step } from "@/app/api";

const reactFlowMock = jest.fn(({ children }: any) => (
  <div data-testid="react-flow">{children}</div>
));
const previewHookState = {
  previewPlan: null as any,
  hoveredGoalSet: null as number | null,
};
let viewportZoom = 1;

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
  useViewport: () => ({ zoom: viewportZoom }),
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
    hoveredGoalSet: previewHookState.hoveredGoalSet,
    goalSteps: [],
    setGoalSteps: jest.fn(),
    focusedPreviewAttribute: null,
    setFocusedPreviewAttribute: jest.fn(),
    setPreviewPlan: jest.fn(),
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
    previewPlan: null,
    diagramContainerRef: { current: null },
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
  }),
}));

jest.mock("./useStepVisibility", () => ({
  useStepVisibility: (steps: Step[]) => ({
    visibleSteps: steps,
    previewStepIds: new Set(steps.map((step) => step.id)),
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
    previewHookState.hoveredGoalSet = null;
    viewportZoom = 1;
  });

  it("renders diagram scaffolding", () => {
    const { getByTestId } = render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[
            { id: "s1", name: "Step 1", type: "service", attributes: {} },
          ]}
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
          goalSteps: [["s1"]],
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[
            { id: "s1", name: "Step 1", type: "service", attributes: {} },
          ]}
        />
      </DiagramSelectionProvider>
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });

  it("renders floating preview hud when preview is active", () => {
    previewHookState.previewPlan = {
      goals: { steps: ["s1"] },
      steps: {
        s1: {
          step: { id: "s1", name: "Step 1", type: "service", attributes: {} },
          inputs: {},
        },
        s2: {
          step: { id: "s2", name: "Step 2", type: "service", attributes: {} },
          inputs: {},
        },
      },
      excluded: [],
    };

    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [["s1"]],
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[
            { id: "s1", name: "Step 1", type: "service", attributes: {} },
          ]}
        />
      </DiagramSelectionProvider>
    );

    expect(
      screen.getByText(`${t("overview.previewLabel")} · ${t("goals.primary")}`)
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("overview.previewGoals", { count: 1 }))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("overview.previewSteps", { count: 1 }))
    ).toBeInTheDocument();
  });

  it("passes calculated edges directly to ReactFlow", () => {
    const initialEdges = [{ id: "e1-2", style: { stroke: "#111111" } }];
    useEdgeCalculationMock.mockReturnValue(initialEdges);

    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[
            { id: "s1", name: "Step 1", type: "service", attributes: {} },
          ]}
        />
      </DiagramSelectionProvider>
    );

    expect(reactFlowMock).toHaveBeenCalled();
    const props = reactFlowMock.mock.calls[0][0];
    expect(props.edges).toBe(initialEdges);
  });

  it("previews hovered branches and returns to the last group on leave", () => {
    const steps: Step[] = [
      {
        id: "source",
        name: "Source",
        type: "service",
        attributes: {
          value: { role: AttributeRole.Output, type: AttributeType.String },
        },
      },
      {
        id: "primary",
        name: "Primary",
        type: "service",
        attributes: {
          value: { role: AttributeRole.Required, type: AttributeType.String },
        },
      },
      { id: "fallback", name: "Fallback", type: "service", attributes: {} },
    ];
    previewHookState.previewPlan = {
      steps: Object.fromEntries(steps.map((step) => [step.id, step])),
    };
    const selection = {
      goalSteps: [["primary"], ["fallback"]],
      setGoalSteps: jest.fn(),
    };
    const view = () => (
      <DiagramSelectionProvider value={selection}>
        <OverviewDiagramView steps={steps} />
      </DiagramSelectionProvider>
    );
    const { rerender } = render(view());
    expect(useEdgeCalculationMock).toHaveBeenLastCalledWith(
      steps,
      new Set(["fallback"]),
      null
    );
    previewHookState.hoveredGoalSet = 0;
    rerender(view());
    expect(useEdgeCalculationMock).toHaveBeenLastCalledWith(
      steps,
      new Set(["primary", "source"]),
      null
    );
    previewHookState.hoveredGoalSet = null;
    rerender(view());
    expect(useEdgeCalculationMock).toHaveBeenLastCalledWith(
      steps,
      new Set(["fallback"]),
      null
    );
    expect(reactFlowMock.mock.calls.at(-1)![0].onPaneClick).toBeUndefined();
    expect(selection.setGoalSteps).not.toHaveBeenCalled();
  });

  it("uses macOS-style trackpad gestures", () => {
    render(
      <DiagramSelectionProvider
        value={{
          goalSteps: [],
          setGoalSteps: jest.fn(),
        }}
      >
        <OverviewDiagramView
          steps={[
            { id: "s1", name: "Step 1", type: "service", attributes: {} },
          ]}
        />
      </DiagramSelectionProvider>
    );

    expect(reactFlowMock).toHaveBeenCalled();
    const props = reactFlowMock.mock.calls[0][0];
    expect(props.panOnScroll).toBe(true);
    expect(props.zoomOnScroll).toBe(false);
    expect(props.zoomOnPinch).toBe(true);
  });
});
