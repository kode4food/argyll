import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import FlowPage from "./FlowPage";

const MockFlowSelector = () => <div>FlowSelector</div>;
MockFlowSelector.displayName = "MockFlowSelector";

const MockFlowDiagram = () => <div>FlowDiagram</div>;
MockFlowDiagram.displayName = "MockFlowDiagram";

jest.mock("../organisms/FlowSelector", () => MockFlowSelector);
jest.mock("./FlowDiagram", () => MockFlowDiagram);
jest.mock("../../store/flowStore", () => {
  const actual = jest.requireActual("../../store/flowStore");
  return {
    ...actual,
    useFlowError: jest.fn(),
    useLoadSteps: jest.fn(() => jest.fn()),
    useLoadFlows: jest.fn(() => jest.fn()),
  };
});

const flowStore = require("../../store/flowStore");

describe("FlowPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    flowStore.useFlowError.mockReturnValue("boom");
    render(<FlowPage />);
    expect(screen.getByText(/Error: boom/)).toBeInTheDocument();
    const retry = screen.getByRole("button", { name: /Retry/ });
    fireEvent.click(retry);
  });

  it("loads flows and steps on mount", () => {
    const loadSteps = jest.fn();
    const loadFlows = jest.fn();
    flowStore.useFlowError.mockReturnValue(null);
    flowStore.useLoadSteps.mockReturnValue(loadSteps);
    flowStore.useLoadFlows.mockReturnValue(loadFlows);

    render(<FlowPage />);

    expect(loadSteps).toHaveBeenCalled();
    expect(loadFlows).toHaveBeenCalled();
    expect(screen.getByText("FlowSelector")).toBeInTheDocument();
    expect(screen.getByText("FlowDiagram")).toBeInTheDocument();
  });
});
