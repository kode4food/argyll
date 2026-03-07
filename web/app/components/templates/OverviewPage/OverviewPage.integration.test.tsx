import { render, screen } from "@testing-library/react";
import OverviewPage from "./OverviewPage";

jest.mock("@/app/components/organisms/FlowSelector", () => ({
  __esModule: true,
  default: () => <div data-testid="flow-selector" />,
}));

jest.mock("@/app/components/templates/OverviewDiagram", () => ({
  __esModule: true,
  default: () => <div data-testid="overview-diagram" />,
}));

jest.mock("@/app/store/flowStore", () => {
  const api = jest.requireActual("@/app/store/flowStore");
  return {
    ...api,
    useFlowError: jest.fn(() => null),
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
    useSteps: jest.fn(() => []),
    useFlows: jest.fn(() => []),
    useFlowsHasMore: jest.fn(() => false),
    useFlowsLoading: jest.fn(() => false),
    useLoadMoreFlows: jest.fn(() => jest.fn()),
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

  it("renders main content", () => {
    render(<OverviewPage />);

    expect(screen.getByTestId("flow-selector")).toBeInTheDocument();
    expect(screen.getByTestId("overview-diagram")).toBeInTheDocument();
  });
});
