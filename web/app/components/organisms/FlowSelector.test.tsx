import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import FlowSelector from "./FlowSelector";

const pushMock = jest.fn();
const subscribeMock = jest.fn();
let eventsMock: any[] = [];

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
    prefetch: jest.fn(),
  }),
  useParams: () => ({}),
  usePathname: () => "/",
}));

jest.mock("../../hooks/useFlowFromUrl", () => ({
  useFlowFromUrl: jest.fn(),
}));

jest.mock("../../hooks/useWebSocketContext", () => ({
  useWebSocketContext: () => ({
    subscribe: subscribeMock,
    events: eventsMock,
  }),
}));

jest.mock("../../contexts/UIContext", () => {
  const actual = jest.requireActual("../../contexts/UIContext");
  return {
    ...actual,
    useUI: () => ({
      showCreateForm: false,
      setShowCreateForm: jest.fn(),
      previewPlan: null,
      updatePreviewPlan: jest.fn(),
      clearPreviewPlan: jest.fn(),
      setSelectedStep: jest.fn(),
      goalStepIds: [],
      setGoalStepIds: jest.fn(),
    }),
    UIProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
  };
});

jest.mock("../../store/flowStore", () => {
  const actual = jest.requireActual("../../store/flowStore");
  return {
    ...actual,
    useFlows: jest.fn(() => []),
    useSelectedFlow: jest.fn(() => null),
    useSteps: jest.fn(() => []),
    useLoadFlows: jest.fn(() => jest.fn()),
    useAddFlow: jest.fn(() => jest.fn()),
    useRemoveFlow: jest.fn(() => jest.fn()),
    useUpdateFlowStatus: jest.fn(() => jest.fn()),
  };
});

jest.mock("../molecules/KeyboardShortcutsModal", () => () => (
  <div>Shortcuts</div>
));
jest.mock("./FlowCreateForm", () => () => <div>FlowCreateForm</div>);

describe("FlowSelector", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    eventsMock = [];
  });

  it("renders and can open dropdown", () => {
    render(<FlowSelector />);
    const button = screen.getByRole("button", { name: /Select Flow/i });
    fireEvent.click(button);
    expect(screen.getByPlaceholderText(/Search flows/)).toBeInTheDocument();
  });

  it("shows new flow button when no selection", () => {
    render(<FlowSelector />);
    expect(
      screen.getByRole("button", { name: /Create New Flow/i })
    ).toBeInTheDocument();
  });

  it("pushes route when selecting a flow from dropdown", () => {
    const useFlows = require("../../store/flowStore").useFlows;
    useFlows.mockReturnValue([
      { id: "wf-1", status: "pending" },
      { id: "wf-2", status: "completed" },
    ]);

    render(<FlowSelector />);
    fireEvent.click(screen.getByRole("button", { name: /Select Flow/i }));
    fireEvent.mouseDown(screen.getByText("wf-1"));

    expect(pushMock).toHaveBeenCalledWith("/flow/wf-1");
  });

  it("subscribes on mount and updates flow status from events", () => {
    const updateFlowStatus = jest.fn();
    const useUpdateFlowStatus =
      require("../../store/flowStore").useUpdateFlowStatus;
    useUpdateFlowStatus.mockReturnValue(updateFlowStatus);
    eventsMock = [
      {
        type: "flow_completed",
        timestamp: Date.now(),
        sequence: 1,
        id: ["flow", "wf-123"],
      },
    ];

    render(<FlowSelector />);

    expect(subscribeMock).toHaveBeenCalledWith({
      event_types: ["flow_started", "flow_completed", "flow_failed"],
    });
    expect(updateFlowStatus).toHaveBeenCalledWith(
      "wf-123",
      "completed",
      expect.any(String)
    );
  });
});
