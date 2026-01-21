import React from "react";
import { render, screen } from "@testing-library/react";
import LiveDiagram from "./LiveDiagram";
import { t } from "@/app/testUtils/i18n";

const sessionMock = {
  selectedFlow: "wf-1" as string | null,
  flowData: null as any,
  executions: [],
  resolvedAttributes: [],
  loading: false,
  flowNotFound: false,
  steps: [],
};

jest.mock("@/app/contexts/FlowSessionContext", () => ({
  __esModule: true,
  useFlowSession: () => sessionMock,
}));

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({
    clearPreviewPlan: jest.fn(),
    setGoalSteps: jest.fn(),
  }),
}));

jest.mock("@/app/components/templates/LiveDiagramView", () => {
  const Mock = () => <div data-testid="live-diagram-view" />;
  Mock.displayName = "MockLiveDiagramView";
  return Mock;
});

jest.mock("@/app/components/organisms/FlowStats", () => {
  const Mock = () => <div data-testid="flow-stats" />;
  Mock.displayName = "MockFlowStats";
  return Mock;
});

describe("LiveDiagram", () => {
  beforeEach(() => {
    sessionMock.selectedFlow = "wf-1";
    sessionMock.flowData = null;
    sessionMock.executions = [];
    sessionMock.resolvedAttributes = [];
    sessionMock.loading = false;
    sessionMock.flowNotFound = false;
    sessionMock.steps = [];
  });

  it("shows not found state when flow missing", () => {
    sessionMock.flowNotFound = true;
    render(<LiveDiagram />);
    expect(screen.getByText(t("live.flowNotFoundTitle"))).toBeInTheDocument();
  });

  it("renders header and stats when flow data is available", () => {
    sessionMock.flowData = {
      id: "wf-1",
      status: "active",
      plan: { steps: { step1: {} } },
      started_at: new Date().toISOString(),
    };
    sessionMock.steps = [{ id: "step1", name: "Step 1", type: "sync" }];

    render(<LiveDiagram />);
    expect(screen.getByText("wf-1")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
    expect(screen.getByTestId("flow-stats")).toBeInTheDocument();
  });

  it("renders live diagram", () => {
    render(<LiveDiagram />);
    expect(screen.getByTestId("live-diagram-view")).toBeInTheDocument();
  });
});
