import React from "react";
import { render, screen } from "@testing-library/react";
import LivePage from "./LivePage";

jest.mock("@/app/components/organisms/FlowSelector", () => {
  const Mock = () => <div>FlowSelector</div>;
  Mock.displayName = "MockFlowSelector";
  return Mock;
});

jest.mock("@/app/components/templates/LiveDiagram", () => {
  const Mock = () => <div>LiveDiagram</div>;
  Mock.displayName = "MockLiveDiagram";
  return Mock;
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

describe("LivePage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    flowSession.flowError = "boom";
    render(<LivePage />);
    expect(screen.getByText(/Error: boom/)).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    flowSession.flowError = null;
    render(<LivePage />);
    expect(screen.getByText("FlowSelector")).toBeInTheDocument();
    expect(screen.getByText("LiveDiagram")).toBeInTheDocument();
  });
});
