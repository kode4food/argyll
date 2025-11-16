import React from "react";
import { render } from "@testing-library/react";
import ConnectionStatusWrapper from "./ConnectionStatusWrapper";
import { useWebSocketContext } from "../../hooks/useWebSocketContext";

jest.mock("../../hooks/useWebSocketContext");
jest.mock("./ConnectionStatusIndicator", () => {
  return function MockConnectionStatusIndicator({
    status,
    reconnectAttempt,
    onReconnect,
  }: any) {
    return (
      <div data-testid="connection-status-indicator">
        <div data-testid="status">{status}</div>
        <div data-testid="reconnect-attempt">{reconnectAttempt}</div>
        <button data-testid="reconnect-button" onClick={onReconnect}>
          Reconnect
        </button>
      </div>
    );
  };
});

const mockUseWebSocketContext = useWebSocketContext as jest.MockedFunction<
  typeof useWebSocketContext
>;

describe("ConnectionStatusWrapper", () => {
  const defaultWebSocketContext = {
    isConnected: true,
    connectionStatus: "connected" as const,
    events: [],
    reconnectAttempt: 0,
    subscribe: jest.fn(),
    reconnect: jest.fn(),
    registerConsumer: jest.fn(),
    unregisterConsumer: jest.fn(),
    updateConsumerCursor: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseWebSocketContext.mockReturnValue(defaultWebSocketContext);
  });

  test("renders ConnectionStatusIndicator with context values", () => {
    const { getByTestId } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("connection-status-indicator")).toBeInTheDocument();
    expect(getByTestId("status")).toHaveTextContent("connected");
    expect(getByTestId("reconnect-attempt")).toHaveTextContent("0");
  });

  test("passes reconnect handler to ConnectionStatusIndicator", () => {
    const mockReconnect = jest.fn();
    mockUseWebSocketContext.mockReturnValue({
      ...defaultWebSocketContext,
      reconnect: mockReconnect,
    });

    const { getByTestId } = render(<ConnectionStatusWrapper />);
    const reconnectButton = getByTestId("reconnect-button");

    reconnectButton.click();

    expect(mockReconnect).toHaveBeenCalled();
  });

  test("updates when connection status changes", () => {
    const { getByTestId, rerender } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("status")).toHaveTextContent("connected");

    mockUseWebSocketContext.mockReturnValue({
      ...defaultWebSocketContext,
      connectionStatus: "reconnecting",
    });

    rerender(<ConnectionStatusWrapper />);

    expect(getByTestId("status")).toHaveTextContent("reconnecting");
  });

  test("updates when reconnect attempt changes", () => {
    const { getByTestId, rerender } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("reconnect-attempt")).toHaveTextContent("0");

    mockUseWebSocketContext.mockReturnValue({
      ...defaultWebSocketContext,
      reconnectAttempt: 3,
    });

    rerender(<ConnectionStatusWrapper />);

    expect(getByTestId("reconnect-attempt")).toHaveTextContent("3");
  });
});
