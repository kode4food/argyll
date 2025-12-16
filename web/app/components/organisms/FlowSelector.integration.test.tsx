import React from "react";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import FlowSelector from "./FlowSelector";
import { UIProvider } from "../../contexts/UIContext";
import { FlowSessionProvider } from "../../contexts/FlowSessionContext";

jest.mock("./FlowSelector/useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("../../hooks/useWebSocketContext", () => ({
  useWebSocketContext: () => ({
    subscribe: jest.fn(),
    events: [],
  }),
}));

jest.mock("../../store/flowStore", () => {
  const loadFlows = jest.fn().mockResolvedValue(undefined);
  const loadSteps = jest.fn().mockResolvedValue(undefined);
  const updateFlowStatus = jest.fn();
  return {
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
    useLoadFlows: jest.fn(() => loadFlows),
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
      { id: "wf-1", status: "pending" },
      { id: "wf-2", status: "completed" },
    ]),
    useAddFlow: jest.fn(() => jest.fn()),
    useRemoveFlow: jest.fn(() => jest.fn()),
    useUpdateFlowStatus: jest.fn(() => updateFlowStatus),
    useFlowData: jest.fn(() => null),
    useFlowLoading: jest.fn(() => false),
    useFlowNotFound: jest.fn(() => false),
    useIsFlowMode: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
    useFlowError: jest.fn(() => null),
    __loadFlows: loadFlows,
    __loadSteps: loadSteps,
    __updateFlowStatus: updateFlowStatus,
  };
});

jest.mock("../../api", () => ({
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

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: jest.fn(),
    prefetch: jest.fn(),
  }),
  useParams: () => ({}),
  usePathname: () => "/",
}));

jest.mock("./FlowCreateForm", () => ({
  __esModule: true,
  default: () => <div>FlowCreateForm</div>,
}));

describe("FlowSelector integration", () => {
  it("loads flows on mount and can open create form", async () => {
    const flowStore = require("../../store/flowStore");
    render(
      <UIProvider>
        <FlowSessionProvider>
          <FlowSelector />
        </FlowSessionProvider>
      </UIProvider>
    );

    await waitFor(() => expect(flowStore.__loadFlows).toHaveBeenCalled());

    act(() => {
      fireEvent.click(screen.getByRole("button", { name: /Create New Flow/i }));
    });

    expect(await screen.findByText(/FlowCreateForm/i)).toBeInTheDocument();
  });
});
