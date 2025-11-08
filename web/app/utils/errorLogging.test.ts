import React from "react";

describe("errorLogger", () => {
  let errorLogger: any;
  let consoleGroupSpy: jest.SpyInstance;
  let consoleErrorSpy: jest.SpyInstance;
  let consoleLogSpy: jest.SpyInstance;
  let consoleGroupEndSpy: jest.SpyInstance;
  let originalEnv: string | undefined;

  beforeAll(() => {
    originalEnv = process.env.NODE_ENV;
    Object.defineProperty(process.env, "NODE_ENV", {
      value: "development",
      writable: true,
      configurable: true,
    });
    jest.resetModules();
    const errorLogging = require("./errorLogging");
    errorLogger = errorLogging.errorLogger;
  });

  afterAll(() => {
    Object.defineProperty(process.env, "NODE_ENV", {
      value: originalEnv,
      writable: true,
      configurable: true,
    });
    jest.resetModules();
  });

  beforeEach(() => {
    consoleGroupSpy = jest.spyOn(console, "group").mockImplementation();
    consoleErrorSpy = jest.spyOn(console, "error").mockImplementation();
    consoleLogSpy = jest.spyOn(console, "log").mockImplementation();
    consoleGroupEndSpy = jest.spyOn(console, "groupEnd").mockImplementation();
  });

  afterEach(() => {
    consoleGroupSpy.mockRestore();
    consoleErrorSpy.mockRestore();
    consoleLogSpy.mockRestore();
    consoleGroupEndSpy.mockRestore();
  });

  describe("logError", () => {
    test("logs basic error", () => {
      const error = new Error("Test error");

      errorLogger.logError(error);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          timestamp: expect.any(String),
          url: expect.any(String),
          userAgent: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("logs error with React ErrorInfo", () => {
      const error = new Error("React component error");
      const errorInfo: React.ErrorInfo = {
        componentStack: "in MyComponent\n  in App",
      };

      errorLogger.logError(error, errorInfo);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Component Stack:",
        "in MyComponent\n  in App"
      );
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          componentStack: "in MyComponent\n  in App",
          timestamp: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("logs error with additional context", () => {
      const error = new Error("Test error");
      const context = { userId: "123", action: "testAction" };

      errorLogger.logError(error, undefined, context);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          userId: "123",
          action: "testAction",
          timestamp: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("includes all context properties with errorInfo and additionalContext", () => {
      const error = new Error("Complex error");
      const errorInfo: React.ErrorInfo = {
        componentStack: "in TestComponent",
      };
      const additionalContext = {
        userId: "user-123",
        workflowId: "wf-456",
      };

      errorLogger.logError(error, errorInfo, additionalContext);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Component Stack:",
        "in TestComponent"
      );
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          componentStack: "in TestComponent",
          userId: "user-123",
          workflowId: "wf-456",
          timestamp: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("includes browser context when available", () => {
      const error = new Error("Browser error");

      errorLogger.logError(error);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          userAgent: expect.any(String),
          url: expect.stringContaining("localhost"),
          timestamp: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("handles errors with no message", () => {
      const error = new Error();

      errorLogger.logError(error);

      expect(consoleGroupSpy).toHaveBeenCalledWith("🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenCalledWith("Error:", error);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Context:",
        expect.objectContaining({
          timestamp: expect.any(String),
        })
      );
      expect(consoleGroupEndSpy).toHaveBeenCalled();
    });

    test("handles multiple error calls independently", () => {
      const error1 = new Error("First error");
      const error2 = new Error("Second error");

      errorLogger.logError(error1);
      errorLogger.logError(error2);

      expect(consoleGroupSpy).toHaveBeenCalledTimes(2);
      expect(consoleErrorSpy).toHaveBeenCalledTimes(4); // 2 calls per error (Error:, Context:)
      expect(consoleGroupSpy).toHaveBeenNthCalledWith(1, "🚨 Error Logged");
      expect(consoleGroupSpy).toHaveBeenNthCalledWith(2, "🚨 Error Logged");
      expect(consoleErrorSpy).toHaveBeenNthCalledWith(1, "Error:", error1);
      expect(consoleErrorSpy).toHaveBeenNthCalledWith(3, "Error:", error2);
      expect(consoleGroupEndSpy).toHaveBeenCalledTimes(2);
    });
  });
});
