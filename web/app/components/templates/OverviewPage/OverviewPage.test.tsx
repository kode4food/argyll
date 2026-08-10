import React from "react";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import OverviewPage from "./OverviewPage";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/components/organisms/FlowSelector", () => {
  const MockFlowSelector = () => <div data-testid="flow-selector" />;
  MockFlowSelector.displayName = "MockFlowSelector";
  return MockFlowSelector;
});

jest.mock("@/app/components/templates/OverviewDiagram", () => {
  const MockOverviewDiagram = () => <div data-testid="overview-diagram" />;
  MockOverviewDiagram.displayName = "MockOverviewDiagram";
  return MockOverviewDiagram;
});

let mockFlowError: string | null = null;

jest.mock("@/app/store/flowStore", () => ({
  useFlowError: () => mockFlowError,
}));

describe("OverviewPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    mockFlowError = "boom";
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );
    expect(
      screen.getByText(t("common.errorMessage", { message: "boom" }))
    ).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    mockFlowError = null;
    render(
      <MemoryRouter>
        <OverviewPage />
      </MemoryRouter>
    );
    expect(screen.getByTestId("flow-selector")).toBeInTheDocument();
    expect(screen.getByTestId("overview-diagram")).toBeInTheDocument();
  });
});
