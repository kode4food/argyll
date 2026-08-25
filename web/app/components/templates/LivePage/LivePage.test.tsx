import React from "react";
import { render, screen } from "@testing-library/react";
import LivePage from "./LivePage";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/components/organisms/FlowSelector", () => {
  const Mock = () => <div data-testid="flow-selector" />;
  Mock.displayName = "MockFlowSelector";
  return Mock;
});

jest.mock("@/app/components/templates/LiveDiagram", () => {
  const Mock = () => <div data-testid="live-diagram" />;
  Mock.displayName = "MockLiveDiagram";
  return Mock;
});

let mockFlowError: string | null = null;

jest.mock("@/app/store/flowStore", () => ({
  useFlowError: () => mockFlowError,
  useSpaces: () => [],
}));

describe("LivePage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders error state", () => {
    mockFlowError = "boom";
    render(<LivePage />);
    expect(
      screen.getByText(t("common.errorMessage", { message: "boom" }))
    ).toBeInTheDocument();
  });

  it("renders selector and diagram", () => {
    mockFlowError = null;
    render(<LivePage />);
    expect(screen.getByTestId("flow-selector")).toBeInTheDocument();
    expect(screen.getByTestId("live-diagram")).toBeInTheDocument();
  });
});
