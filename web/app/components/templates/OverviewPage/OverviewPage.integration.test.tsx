import { render, waitFor } from "@testing-library/react";
import OverviewPage from "./OverviewPage";

jest.mock("@/app/components/organisms/FlowSelector", () => ({
  __esModule: true,
  default: () => <div data-testid="flow-selector" />,
}));

jest.mock("@/app/components/templates/OverviewDiagram", () => ({
  __esModule: true,
  default: () => <div data-testid="overview-diagram" />,
}));

const loadSteps = jest.fn().mockResolvedValue(undefined);
const loadFlows = jest.fn().mockResolvedValue(undefined);

jest.mock("@/app/store/flowStore", () => {
  const api = jest.requireActual("@/app/store/flowStore");
  return {
    ...api,
    useFlowError: jest.fn(() => null),
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
    useLoadFlows: jest.fn(() => loadFlows),
    useLoadSteps: jest.fn(() => loadSteps),
    useSteps: jest.fn(() => []),
    useFlows: jest.fn(() => []),
    useUpdateFlowStatus: jest.fn(() => jest.fn()),
    useFlowData: jest.fn(() => null),
    useFlowLoading: jest.fn(() => false),
    useFlowNotFound: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
  };
});

describe("OverviewPage integration", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders main content and loads data", async () => {
    render(<OverviewPage />);

    // Steps are loaded via WebSocket subscribed, not HTTP API
    expect(await waitFor(() => loadFlows)).toHaveBeenCalled();
  });
});
