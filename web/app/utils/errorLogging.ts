import React from "react";

interface ErrorContext {
  componentStack?: string;
  userAgent?: string;
  timestamp: string;
  url?: string;
}

export interface ErrorLog {
  error: Error;
  errorInfo?: React.ErrorInfo;
  context?: ErrorContext;
}

class ErrorLogger {
  private isDevelopment = process.env.NODE_ENV === "development";

  logError(
    error: Error,
    errorInfo?: React.ErrorInfo,
    additionalContext?: Record<string, any>
  ) {
    const context: ErrorContext = {
      componentStack: errorInfo?.componentStack ?? undefined,
      userAgent:
        typeof window !== "undefined" ? window.navigator.userAgent : undefined,
      timestamp: new Date().toISOString(),
      url: typeof window !== "undefined" ? window.location.href : undefined,
      ...additionalContext,
    };

    const errorLog: ErrorLog = {
      error,
      errorInfo,
      context,
    };

    if (this.isDevelopment) {
      console.group("🚨 Error Logged");
      console.error("Error:", error);
      if (errorInfo) {
        console.error("Component Stack:", errorInfo.componentStack);
      }
      console.error("Context:", context);
      console.groupEnd();
    }

    if (typeof window !== "undefined") {
      this.sendToMonitoringService(errorLog);
    }
  }

  private sendToMonitoringService(_: ErrorLog) {
    // TODO: Implement actual monitoring service integration
    // For now, this is a no-op placeholder
  }
}

export const errorLogger = new ErrorLogger();
