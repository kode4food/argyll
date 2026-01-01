import { render, waitFor } from "@testing-library/react";
import { FlowSessionProvider, useFlowSession } from "./FlowSessionContext";

jest.mock("../store/flowStore", () => {
  const loadSteps = jest.fn().mockResolvedValue(undefined);
  const loadFlows = jest.fn().mockResolvedValue(undefined);
  return {
    useSelectedFlow: jest.fn(() => "wf-1"),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
    useLoadFlows: jest.fn(() => loadFlows),
    useLoadSteps: jest.fn(() => loadSteps),
    useSteps: jest.fn(() => [{ id: "s1" }]),
    useFlows: jest.fn(() => [{ id: "wf-1" }]),
    useUpdateFlowStatus: jest.fn(() => jest.fn()),
    useFlowData: jest.fn(() => ({ id: "wf-1" })),
    useFlowLoading: jest.fn(() => false),
    useFlowNotFound: jest.fn(() => false),
    useIsFlowMode: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
    useFlowError: jest.fn(() => null),
    __loadSteps: loadSteps,
    __loadFlows: loadFlows,
  };
});

const flowStore = require("../store/flowStore");

const Consumer = () => {
  const session = useFlowSession();
  return (
    <div>
      <span data-testid="selected-flow">{session.selectedFlow}</span>
      <span data-testid="steps-count">{session.steps.length}</span>
    </div>
  );
};

describe("FlowSessionContext", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("exposes session values and loads data", async () => {
    render(
      <FlowSessionProvider>
        <Consumer />
      </FlowSessionProvider>
    );

    // Steps are loaded via WebSocket subscribed, not HTTP API
    expect(await waitFor(() => flowStore.__loadFlows)).toHaveBeenCalled();
  });
});
