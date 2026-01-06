import { render } from "@testing-library/react";
import ConnectionStatusWrapper from "./ConnectionStatusWrapper";
import {
  useEngineConnectionStatus,
  useEngineReconnectAttempt,
  useRequestEngineReconnect,
} from "@/app/store/flowStore";

jest.mock("@/app/store/flowStore", () => ({
  useEngineConnectionStatus: jest.fn(),
  useEngineReconnectAttempt: jest.fn(),
  useRequestEngineReconnect: jest.fn(),
}));

jest.mock("@/app/components/atoms/ConnectionStatusIndicator", () => {
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

const mockUseEngineConnectionStatus =
  useEngineConnectionStatus as jest.MockedFunction<
    typeof useEngineConnectionStatus
  >;
const mockUseEngineReconnectAttempt =
  useEngineReconnectAttempt as jest.MockedFunction<
    typeof useEngineReconnectAttempt
  >;
const mockUseRequestEngineReconnect =
  useRequestEngineReconnect as jest.MockedFunction<
    typeof useRequestEngineReconnect
  >;

describe("ConnectionStatusWrapper", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseEngineConnectionStatus.mockReturnValue("connected");
    mockUseEngineReconnectAttempt.mockReturnValue(0);
    mockUseRequestEngineReconnect.mockReturnValue(jest.fn());
  });

  test("renders ConnectionStatusIndicator with store values", () => {
    const { getByTestId } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("connection-status-indicator")).toBeInTheDocument();
    expect(getByTestId("status")).toHaveTextContent("connected");
    expect(getByTestId("reconnect-attempt")).toHaveTextContent("0");
  });

  test("passes reconnect handler to ConnectionStatusIndicator", () => {
    const mockReconnect = jest.fn();
    mockUseRequestEngineReconnect.mockReturnValue(mockReconnect);

    const { getByTestId } = render(<ConnectionStatusWrapper />);
    const reconnectButton = getByTestId("reconnect-button");

    reconnectButton.click();

    expect(mockReconnect).toHaveBeenCalled();
  });

  test("updates when connection status changes", () => {
    const { getByTestId, rerender } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("status")).toHaveTextContent("connected");

    mockUseEngineConnectionStatus.mockReturnValue("reconnecting");
    rerender(<ConnectionStatusWrapper />);

    expect(getByTestId("status")).toHaveTextContent("reconnecting");
  });

  test("updates when reconnect attempt changes", () => {
    const { getByTestId, rerender } = render(<ConnectionStatusWrapper />);

    expect(getByTestId("reconnect-attempt")).toHaveTextContent("0");

    mockUseEngineReconnectAttempt.mockReturnValue(3);
    rerender(<ConnectionStatusWrapper />);

    expect(getByTestId("reconnect-attempt")).toHaveTextContent("3");
  });
});
