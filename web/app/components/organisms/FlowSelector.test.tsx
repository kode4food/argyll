import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import FlowSelector from "./FlowSelector";

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: jest.fn(),
    prefetch: jest.fn(),
  }),
  useParams: () => ({}),
  usePathname: () => "/",
}));

jest.mock("../../hooks/useWebSocketContext", () => ({
  useWebSocketContext: () => ({
    subscribe: jest.fn(),
    events: [],
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
});
