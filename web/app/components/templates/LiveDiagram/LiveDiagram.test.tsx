import React from "react";
import { act, render, screen } from "@testing-library/react";
import LiveDiagram from "./LiveDiagram";
import { useChatStore } from "@/app/store/chatStore";
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

jest.mock("@/app/store/flowStore", () => ({
  useSelectedFlow: () => sessionMock.selectedFlow,
  useFlowData: () => sessionMock.flowData,
  useExecutions: () => sessionMock.executions,
  useResolvedAttributes: () => sessionMock.resolvedAttributes,
  useFlowLoading: () => sessionMock.loading,
  useFlowNotFound: () => sessionMock.flowNotFound,
  useSteps: () => sessionMock.steps,
}));

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({
    clearPreviewPlan: jest.fn(),
    setGoalSteps: jest.fn(),
  }),
}));

const mockNavigate = jest.fn();
jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => mockNavigate,
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
    sessionMock.steps = [{ id: "step1", name: "Step 1", type: "service" }];

    render(<LiveDiagram />);
    expect(screen.getByText("wf-1")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
    expect(screen.getByTestId("flow-stats")).toBeInTheDocument();
  });

  it("renders live diagram", () => {
    render(<LiveDiagram />);
    expect(screen.getByTestId("live-diagram-view")).toBeInTheDocument();
  });

  describe("chat actions", () => {
    beforeEach(() => {
      mockNavigate.mockClear();
    });

    afterEach(() => {
      useChatStore.setState({ action: null });
    });

    it("opens a Flow", () => {
      render(<LiveDiagram />);
      act(() => {
        useChatStore.setState({ action: { type: "open_flow", flow_id: "f2" } });
      });
      expect(mockNavigate).toHaveBeenCalledWith("/flow/f2");
      expect(useChatStore.getState().action).toBeNull();
    });

    it("stays on the Flow it already shows", () => {
      render(<LiveDiagram />);
      act(() => {
        useChatStore.setState({
          action: { type: "open_flow", flow_id: "wf-1" },
        });
      });
      expect(mockNavigate).not.toHaveBeenCalled();
      expect(useChatStore.getState().action).toBeNull();
    });

    it("leaves other actions for the overview", () => {
      render(<LiveDiagram />);
      act(() => {
        useChatStore.setState({ action: { type: "open_step", step_id: "a" } });
      });
      expect(mockNavigate).toHaveBeenCalledWith("/");
      expect(useChatStore.getState().action).toEqual({
        type: "open_step",
        step_id: "a",
      });
    });
  });
});
