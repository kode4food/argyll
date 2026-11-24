import { act, fireEvent, render, screen } from "@testing-library/react";
import { useAuthStore } from "@/app/store/authStore";
import { t } from "@/app/testUtils/i18n";
import AuthGate from "./AuthGate";

describe("AuthGate", () => {
  const login = jest.fn().mockResolvedValue(undefined);

  beforeEach(() => jest.clearAllMocks());

  test("renders children once signed in", async () => {
    useAuthStore.setState({
      isReady: true,
      start: jest.fn().mockResolvedValue(undefined),
      login,
    });
    await act(async () => {
      render(<AuthGate>app</AuthGate>);
    });
    expect(screen.getByText("app")).toBeInTheDocument();
  });

  test("waits while signing in", async () => {
    useAuthStore.setState({
      isReady: false,
      start: jest.fn().mockResolvedValue(undefined),
      login,
    });
    await act(async () => {
      render(<AuthGate>app</AuthGate>);
    });
    expect(screen.getByRole("status")).toHaveAccessibleName(
      t("auth.signingIn")
    );
    expect(screen.queryByText("app")).not.toBeInTheDocument();
  });

  test("offers a retry when sign-in fails", async () => {
    useAuthStore.setState({
      isReady: false,
      start: jest.fn().mockRejectedValue(new Error("bad state")),
      login,
    });
    await act(async () => {
      render(<AuthGate>app</AuthGate>);
    });
    expect(screen.getByText(t("auth.failedTitle"))).toBeInTheDocument();

    fireEvent.click(screen.getByText(t("common.tryAgain")));
    expect(login).toHaveBeenCalled();
  });
});
