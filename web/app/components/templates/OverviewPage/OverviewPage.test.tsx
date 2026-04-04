import React from "react";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import OverviewPage from "./OverviewPage";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/components/organisms/FlowSelector", () => {
  const MockFlowSelector = () => <div data-testid="flow-selector" />;
  MockFlowSelector.displayName = "MockFlowSelector";
  return MockFlowSelector;
});

jest.mock("@/app/components/templates/OverviewDiagram", () => {
  const MockOverviewDiagram = () => <div data-testid="overview-diagram" />;
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
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );
    expect(
      screen.getByText(t("common.errorMessage", { message: "boom" }))
    ).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    flowSession.flowError = null;
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );
    expect(screen.getByTestId("flow-selector")).toBeInTheDocument();
    expect(screen.getByTestId("overview-diagram")).toBeInTheDocument();
  });
});
