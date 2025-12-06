import React from "react";
import { render } from "@testing-library/react";

import StepDiagram from "./StepDiagram";

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

jest.mock("@/utils/nodePositioning", () => ({
  calculateNodePositions: jest.fn(() => ({
    nodes: [],
    edges: [],
  })),
  loadNodePositions: jest.fn(() => ({ nodes: [], edges: [] })),
  saveNodePositions: jest.fn(),
}));

jest.mock("@/utils/stepUtils", () => ({
  sortStepsByType: jest.fn((steps) => steps),
}));

jest.mock("../../contexts/UIContext", () => ({
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useUI: () => ({
    setSelectedStep: jest.fn(),
    selectedStep: null,
    disableEdit: false,
    diagramContainerRef: { current: null },
  }),
}));

describe("StepDiagram", () => {
  it("renders diagram scaffolding", () => {
    const { getByTestId } = render(
      <StepDiagram
        steps={[
          { id: "s1", name: "Step 1", type: "sync", attributes: {} } as any,
        ]}
        selectedStep={null}
        onSelectStep={jest.fn()}
        flowData={null}
        executions={[]}
        resolvedAttributes={[]}
      />
    );

    expect(getByTestId("react-flow")).toBeInTheDocument();
  });
});
