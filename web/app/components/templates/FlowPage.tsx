"use client";

import React from "react";
import FlowDiagram from "./FlowDiagram";
import FlowSelector from "../organisms/FlowSelector";
import ErrorBoundary from "../organisms/ErrorBoundary";
import { UIProvider } from "../../contexts/UIContext";
import {
  FlowSessionProvider,
  useFlowSession,
} from "../../contexts/FlowSessionContext";
import styles from "./FlowPage.module.css";

function FlowPageContent() {
  return (
    <div className={styles.page}>
      <ErrorBoundary
        title="Flow Selector Error"
        description="An error occurred in the flow selector. Try refreshing the page."
      >
        <FlowSelector />
      </ErrorBoundary>
      <div className={styles.mainContent}>
        <ErrorBoundary
          title="Diagram Error"
          description="An error occurred while rendering the diagram. Try selecting a different flow."
        >
          <FlowDiagram />
        </ErrorBoundary>
      </div>
    </div>
  );
}

function FlowPageWithSession() {
  const { flowError } = useFlowSession();

  if (flowError) {
    return (
      <div className={styles.errorPage}>
        <div className={styles.errorContent}>
          <p className={styles.errorMessage}>Error: {flowError}</p>
          <button
            onClick={() => window.location.reload()}
            className={styles.retryButton}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return <FlowPageContent />;
}

export default function FlowPage() {
  return (
    <UIProvider>
      <FlowSessionProvider>
        <FlowPageWithSession />
      </FlowSessionProvider>
    </UIProvider>
  );
}
