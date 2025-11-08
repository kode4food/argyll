"use client";

import React, { Component, ReactNode } from "react";
import ErrorFallback from "../molecules/ErrorFallback";
import { errorLogger } from "../../utils/errorLogging";

interface Props {
  children: ReactNode;
  fallback?: (error: Error, resetError: () => void) => ReactNode;
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  title?: string;
  description?: string;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    errorLogger.logError(error, errorInfo, {
      boundaryTitle: this.props.title,
      boundaryDescription: this.props.description,
    });

    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }
  }

  resetError = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError && this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.resetError);
      }

      return (
        <ErrorFallback
          error={this.state.error}
          resetError={this.resetError}
          title={this.props.title}
          description={this.props.description}
        />
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
