import React from "react";
import { render, screen } from "@testing-library/react";
import OverviewPage from "./OverviewPage";

jest.mock("@/app/components/organisms/FlowSelector", () => {
  const MockFlowSelector = () => <div>FlowSelector</div>;
  MockFlowSelector.displayName = "MockFlowSelector";
  return MockFlowSelector;
});

jest.mock("@/app/components/templates/OverviewDiagram", () => {
  const MockOverviewDiagram = () => <div>OverviewDiagram</div>;
  MockOverviewDiagram.displayName = "MockOverviewDiagram";
  return MockOverviewDiagram;
});

jest.mock("@/app/contexts/FlowSessionContext", () => {
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
} = require("@/app/contexts/FlowSessionContext");

describe("OverviewPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    flowSession.flowError = "boom";
    render(<OverviewPage />);
    expect(screen.getByText(/Error: boom/)).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    flowSession.flowError = null;
    render(<OverviewPage />);
    expect(screen.getByText("FlowSelector")).toBeInTheDocument();
    expect(screen.getByText("OverviewDiagram")).toBeInTheDocument();
  });
});
