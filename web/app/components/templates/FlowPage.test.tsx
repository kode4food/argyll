import React from "react";
import { render, screen } from "@testing-library/react";
import FlowPage from "./FlowPage";

jest.mock("../organisms/FlowSelector", () => {
  const MockFlowSelector = () => <div>FlowSelector</div>;
  MockFlowSelector.displayName = "MockFlowSelector";
  return MockFlowSelector;
});

jest.mock("./FlowDiagram", () => {
  const MockFlowDiagram = () => <div>FlowDiagram</div>;
  MockFlowDiagram.displayName = "MockFlowDiagram";
  return MockFlowDiagram;
});

jest.mock("../../contexts/FlowSessionContext", () => {
  const session = {
    selectedFlow: null,
    selectFlow: jest.fn(),
    loadFlows: jest.fn(),
    loadSteps: jest.fn(),
    steps: [],
    flows: [],
    updateFlowStatus: jest.fn(),
    flowData: null,
    loading: false,
    flowNotFound: false,
    isFlowMode: false,
    executions: [],
    resolvedAttributes: [],
    flowError: null as string | null,
  };
  return {
    __esModule: true,
    FlowSessionProvider: ({ children }: { children: React.ReactNode }) =>
      children,
    useFlowSession: () => session,
    __sessionMock: session,
  };
});

const {
  __sessionMock: flowSession,
} = require("../../contexts/FlowSessionContext");

describe("FlowPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    flowSession.flowError = "boom";
    render(<FlowPage />);
    expect(screen.getByText(/Error: boom/)).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    flowSession.flowError = null;
    render(<FlowPage />);
    expect(screen.getByText("FlowSelector")).toBeInTheDocument();
    expect(screen.getByText("FlowDiagram")).toBeInTheDocument();
  });
});
