import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import FlowSelector from "./FlowSelector";
import { UIProvider } from "@/app/contexts/UIContext";
import { FlowSessionProvider } from "@/app/contexts/FlowSessionContext";
import { t } from "@/app/testUtils/i18n";

jest.mock("./useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("@/app/store/flowStore", () => {
  const loadFlows = jest.fn().mockResolvedValue(undefined);
  const loadSteps = jest.fn().mockResolvedValue(undefined);
  const setVisibleFlowIDs = jest.fn();
  return {
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
    useLoadFlows: jest.fn(() => loadFlows),
    useSetVisibleFlowIDs: jest.fn(() => setVisibleFlowIDs),
    useLoadSteps: jest.fn(() => loadSteps),
    useSteps: jest.fn(() => [
      {
        id: "goal",
        name: "Goal",
        type: "script",
        attributes: {},
        script: { language: "lua", script: "" },
      },
    ]),
    useFlows: jest.fn(() => [
      { id: "wf-1", status: "pending", timestamp: "2024-01-01T00:00:00Z" },
      { id: "wf-2", status: "completed", timestamp: "2024-01-02T00:00:00Z" },
    ]),
    useFlowsHasMore: jest.fn(() => false),
    useFlowsLoading: jest.fn(() => false),
    useLoadMoreFlows: jest.fn(() => jest.fn()),
    useAddFlow: jest.fn(() => jest.fn()),
    useRemoveFlow: jest.fn(() => jest.fn()),
    useFlowData: jest.fn(() => null),
    useFlowLoading: jest.fn(() => false),
    useFlowNotFound: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
    useFlowError: jest.fn(() => null),
    __loadFlows: loadFlows,
    __loadSteps: loadSteps,
    __setVisibleFlowIDs: setVisibleFlowIDs,
  };
});

jest.mock("@/app/api", () => ({
  api: {
    getExecutionPlan: jest.fn().mockResolvedValue({
      steps: { goal: {} },
      required: [],
    }),
    startFlow: jest.fn().mockResolvedValue({
      id: "new-flow",
      status: "pending",
    }),
  },
}));

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => jest.fn(),
}));

jest.mock("../FlowCreateForm/FlowCreateForm", () => ({
  __esModule: true,
  default: () => <div data-testid="flow-create-form" />,
}));

describe("FlowSelector integration", () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("loads flows when the dropdown opens and can open create form", async () => {
    const flowStore = require("@/app/store/flowStore");
    render(
      <UIProvider>
        <FlowSessionProvider>
          <FlowSelector />
        </FlowSessionProvider>
      </UIProvider>
    );

    expect(flowStore.__loadFlows).not.toHaveBeenCalled();

    act(() => {
      fireEvent.click(
        screen.getByRole("button", { name: t("flowSelector.selectFlow") })
      );
    });

    await waitFor(() => expect(flowStore.__loadFlows).toHaveBeenCalled());
    expect(flowStore.__setVisibleFlowIDs).not.toHaveBeenCalledWith([
      "wf-1",
      "wf-2",
    ]);

    act(() => {
      jest.advanceTimersByTime(150);
    });

    expect(flowStore.__setVisibleFlowIDs).toHaveBeenLastCalledWith([
      "wf-1",
      "wf-2",
    ]);

    act(() => {
      fireEvent.click(
        screen.getByRole("button", { name: t("flowSelector.createNewFlow") })
      );
    });

    expect(await screen.findByTestId("flow-create-form")).toBeInTheDocument();
  });
});
