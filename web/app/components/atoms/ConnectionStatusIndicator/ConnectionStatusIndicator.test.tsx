import { render, screen, fireEvent } from "@testing-library/react";
import ConnectionStatusIndicator from "./ConnectionStatusIndicator";
import { t } from "@/app/testUtils/i18n";

describe("ConnectionStatusIndicator", () => {
  test("renders nothing when connected", () => {
    const { container } = render(
      <ConnectionStatusIndicator status="connected" />
    );
    expect(container.firstChild).toBeNull();
  });

  test("renders connecting status", () => {
    render(<ConnectionStatusIndicator status="connecting" />);
    expect(
      screen.getByText(t("connectionStatus.connecting"))
    ).toBeInTheDocument();
  });

  test("renders reconnecting status with attempt number", () => {
    render(
      <ConnectionStatusIndicator status="reconnecting" reconnectAttempt={3} />
    );
    expect(
      screen.getByText(t("connectionStatus.reconnecting", { attempt: 3 }))
    ).toBeInTheDocument();
  });

  test("renders disconnected status", () => {
    render(<ConnectionStatusIndicator status="disconnected" />);
    expect(
      screen.getByText(t("connectionStatus.disconnected"))
    ).toBeInTheDocument();
  });

  test("renders failed status", () => {
    render(<ConnectionStatusIndicator status="failed" />);
    expect(screen.getByText(t("connectionStatus.failed"))).toBeInTheDocument();
  });

  test("shows retry button when disconnected with onReconnect", () => {
    const mockReconnect = jest.fn();
    render(
      <ConnectionStatusIndicator
        status="disconnected"
        onReconnect={mockReconnect}
      />
    );
    expect(
      screen.getByRole("button", { name: t("common.retry") })
    ).toBeInTheDocument();
  });

  test("shows retry button when failed with onReconnect", () => {
    const mockReconnect = jest.fn();
    render(
      <ConnectionStatusIndicator status="failed" onReconnect={mockReconnect} />
    );
    expect(
      screen.getByRole("button", { name: t("common.retry") })
    ).toBeInTheDocument();
  });

  test("does not show retry button when no onReconnect provided", () => {
    render(<ConnectionStatusIndicator status="disconnected" />);
    expect(
      screen.queryByRole("button", { name: t("common.retry") })
    ).not.toBeInTheDocument();
  });

  test("calls onReconnect when retry button clicked", () => {
    const mockReconnect = jest.fn();
    render(
      <ConnectionStatusIndicator
        status="disconnected"
        onReconnect={mockReconnect}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: t("common.retry") }));
    expect(mockReconnect).toHaveBeenCalledTimes(1);
  });

  test("does not show retry button when connecting", () => {
    const mockReconnect = jest.fn();
    render(
      <ConnectionStatusIndicator
        status="connecting"
        onReconnect={mockReconnect}
      />
    );
    expect(
      screen.queryByRole("button", { name: t("common.retry") })
    ).not.toBeInTheDocument();
  });

  test("defaults reconnectAttempt to 0", () => {
    render(<ConnectionStatusIndicator status="reconnecting" />);
    expect(
      screen.getByText(t("connectionStatus.reconnecting", { attempt: 0 }))
    ).toBeInTheDocument();
  });

  test("handles unknown status as connected", () => {
    render(
      <ConnectionStatusIndicator
        status={
          "unknown" as unknown as Parameters<
            typeof ConnectionStatusIndicator
          >[0]["status"]
        }
      />
    );
    expect(
      screen.getByText(t("connectionStatus.connected"))
    ).toBeInTheDocument();
  });
});
