import React from "react";
import { render, screen } from "@testing-library/react";
import ErrorBoundary from "./ErrorBoundary";
import { errorLogger } from "../../utils/errorLogging";

jest.mock("../../utils/errorLogging", () => ({
  errorLogger: {
    logError: jest.fn(),
  },
}));

jest.mock("../molecules/ErrorFallback", () => ({
  __esModule: true,
  default: ({ error, resetError, title, description }: any) => (
    <div data-testid="error-fallback">
      <div>Error: {error.message}</div>
      {title && <div>Title: {title}</div>}
      {description && <div>Description: {description}</div>}
      <button onClick={resetError}>Reset</button>
    </div>
  ),
}));

const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
  if (shouldThrow) {
    throw new Error("Test error");
  }
  return <div>No error</div>;
};

describe("ErrorBoundary", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    (console.error as jest.Mock).mockRestore();
  });

  test("renders children when no error", () => {
    render(
      <ErrorBoundary>
        <div>Test content</div>
      </ErrorBoundary>
    );

    expect(screen.getByText("Test content")).toBeInTheDocument();
  });

  test("catches error and shows fallback", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByTestId("error-fallback")).toBeInTheDocument();
    expect(screen.getByText("Error: Test error")).toBeInTheDocument();
  });

  test("logs error with errorLogger", () => {
    render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(errorLogger.logError).toHaveBeenCalled();
    const call = (errorLogger.logError as jest.Mock).mock.calls[0];
    expect(call[0].message).toBe("Test error");
  });

  test("passes title and description to fallback", () => {
    render(
      <ErrorBoundary title="Custom Title" description="Custom Description">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText("Title: Custom Title")).toBeInTheDocument();
    expect(
      screen.getByText("Description: Custom Description")
    ).toBeInTheDocument();
  });

  test("calls onError callback when error occurs", () => {
    const onError = jest.fn();

    render(
      <ErrorBoundary onError={onError}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(onError).toHaveBeenCalled();
    const call = onError.mock.calls[0];
    expect(call[0].message).toBe("Test error");
  });

  test("uses custom fallback when provided", () => {
    const customFallback = (error: Error, resetError: () => void) => (
      <div data-testid="custom-fallback">
        <div>Custom: {error.message}</div>
        <button onClick={resetError}>Custom Reset</button>
      </div>
    );

    render(
      <ErrorBoundary fallback={customFallback}>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByTestId("custom-fallback")).toBeInTheDocument();
    expect(screen.getByText("Custom: Test error")).toBeInTheDocument();
  });

  test("resets error when resetError called", () => {
    const { rerender } = render(
      <ErrorBoundary>
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByTestId("error-fallback")).toBeInTheDocument();

    const resetButton = screen.getByText("Reset");
    resetButton.click();

    rerender(
      <ErrorBoundary>
        <ThrowError shouldThrow={false} />
      </ErrorBoundary>
    );

    expect(screen.queryByTestId("error-fallback")).not.toBeInTheDocument();
    expect(screen.getByText("No error")).toBeInTheDocument();
  });

  test("logs boundary context with error", () => {
    render(
      <ErrorBoundary title="Boundary Title" description="Boundary Description">
        <ThrowError shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(errorLogger.logError).toHaveBeenCalled();
    const call = (errorLogger.logError as jest.Mock).mock.calls[0];
    expect(call[2]).toEqual({
      boundaryTitle: "Boundary Title",
      boundaryDescription: "Boundary Description",
    });
  });
});
