import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import App from "./App";

jest.mock("./contexts/WebSocketProvider", () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="websocket-provider">{children}</div>
  ),
}));

jest.mock("./components/atoms/ConnectionStatusWrapper", () => ({
  __esModule: true,
  default: () => <div data-testid="connection-status-wrapper" />,
}));

jest.mock("./components/templates/OverviewPage", () => ({
  __esModule: true,
  default: () => (
    <div data-testid="overview-page">
      <input data-testid="overview-input" />
    </div>
  ),
}));

jest.mock("./components/templates/LivePage", () => ({
  __esModule: true,
  default: () => <div data-testid="live-page" />,
}));

jest.mock("./components/organisms/NotFoundPage", () => ({
  __esModule: true,
  default: () => <div data-testid="not-found-page" />,
}));

jest.mock("react-hot-toast", () => ({
  Toaster: () => <div data-testid="toaster" />,
}));

describe("App", () => {
  const renderAt = (path: string) => {
    window.history.pushState({}, "", path);
    return render(<App />);
  };

  afterEach(() => {
    window.history.pushState({}, "", "/");
  });

  test("renders OverviewPage for root route", () => {
    renderAt("/");
    expect(screen.getByTestId("overview-page")).toBeInTheDocument();
    expect(screen.getByTestId("connection-status-wrapper")).toBeInTheDocument();
    expect(screen.getByTestId("toaster")).toBeInTheDocument();
  });

  test("applies autofill suppression attributes on mousedown", () => {
    renderAt("/");

    const overviewInput = screen.getByTestId("overview-input");
    expect(overviewInput).not.toHaveAttribute("autocomplete");
    fireEvent.mouseDown(overviewInput);
    expect(overviewInput).toHaveAttribute("autocomplete", "off");
    expect(overviewInput).toHaveAttribute("data-1p-ignore", "true");
    expect(overviewInput).toHaveAttribute("data-lpignore", "true");
    expect(overviewInput).toHaveAttribute("data-bwignore", "true");

    const dynamicInput = document.createElement("input");
    document.body.appendChild(dynamicInput);
    fireEvent.mouseDown(dynamicInput);
    expect(dynamicInput).toHaveAttribute("autocomplete", "off");
    expect(dynamicInput).toHaveAttribute("data-1p-ignore", "true");
    expect(dynamicInput).toHaveAttribute("data-lpignore", "true");
    expect(dynamicInput).toHaveAttribute("data-bwignore", "true");

    dynamicInput.remove();
  });

  test("renders LivePage for flow route", () => {
    renderAt("/flow/flow-123");
    expect(screen.getByTestId("live-page")).toBeInTheDocument();
  });

  test("renders NotFoundPage for unknown route", () => {
    renderAt("/missing");
    expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
  });
});
