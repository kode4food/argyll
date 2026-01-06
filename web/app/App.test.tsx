import React from "react";
import { render, screen } from "@testing-library/react";
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

jest.mock("./components/templates/FlowPage", () => ({
  __esModule: true,
  default: () => <div data-testid="flow-page" />,
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

  test("renders FlowPage for root route", () => {
    renderAt("/");
    expect(screen.getByTestId("flow-page")).toBeInTheDocument();
    expect(screen.getByTestId("connection-status-wrapper")).toBeInTheDocument();
    expect(screen.getByTestId("toaster")).toBeInTheDocument();
  });

  test("renders FlowPage for flow route", () => {
    renderAt("/flow/flow-123");
    expect(screen.getByTestId("flow-page")).toBeInTheDocument();
  });

  test("renders NotFoundPage for unknown route", () => {
    renderAt("/missing");
    expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
  });
});
