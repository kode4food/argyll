import { render, screen, fireEvent } from "@testing-library/react";
import ErrorFallback from "./ErrorFallback";
import { t } from "@/app/testUtils/i18n";

describe("ErrorFallback", () => {
  const mockResetError = jest.fn();
  const mockError = new Error("Test error message");
  mockError.stack = "Error stack trace here";

  beforeEach(() => {
    mockResetError.mockClear();
  });

  test("renders default title and description", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    expect(screen.getByText(t("errorFallback.title"))).toBeInTheDocument();
    expect(
      screen.getByText(t("errorFallback.description"))
    ).toBeInTheDocument();
  });

  test("renders custom title", () => {
    render(
      <ErrorFallback
        error={mockError}
        resetError={mockResetError}
        title="Custom Error Title"
      />
    );
    expect(screen.getByText("Custom Error Title")).toBeInTheDocument();
  });

  test("renders custom description", () => {
    render(
      <ErrorFallback
        error={mockError}
        resetError={mockResetError}
        description="Custom description here"
      />
    );
    expect(screen.getByText("Custom description here")).toBeInTheDocument();
  });

  test("displays error message in details", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    expect(screen.getByText(/Test error message/)).toBeInTheDocument();
  });

  test("displays error stack in details", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    expect(screen.getByText(/Error stack trace/)).toBeInTheDocument();
  });

  test("renders try again button", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    expect(
      screen.getByRole("button", { name: t("common.tryAgain") })
    ).toBeInTheDocument();
  });

  test("calls resetError when button clicked", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    fireEvent.click(screen.getByRole("button", { name: t("common.tryAgain") }));
    expect(mockResetError).toHaveBeenCalledTimes(1);
  });

  test("renders error details summary", () => {
    render(<ErrorFallback error={mockError} resetError={mockResetError} />);
    expect(screen.getByText(t("errorFallback.details"))).toBeInTheDocument();
  });

  test("handles error without stack", () => {
    const errorNoStack = new Error("Simple error");
    errorNoStack.stack = undefined;
    render(<ErrorFallback error={errorNoStack} resetError={mockResetError} />);
    expect(screen.getByText(/Simple error/)).toBeInTheDocument();
  });
});
