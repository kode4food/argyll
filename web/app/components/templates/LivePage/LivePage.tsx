import React from "react";
import LiveDiagram from "@/app/components/templates/LiveDiagram";
import FlowSelector from "@/app/components/organisms/FlowSelector";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { UIProvider } from "@/app/contexts/UIContext";
import {
  FlowSessionProvider,
  useFlowSession,
} from "@/app/contexts/FlowSessionContext";
import styles from "@/app/components/templates/OverviewPage/OverviewPage.module.css";

function LivePageContent() {
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
          <LiveDiagram />
        </ErrorBoundary>
      </div>
    </div>
  );
}

function LivePageWithSession() {
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

  return <LivePageContent />;
}

export default function LivePage() {
  return (
    <UIProvider>
      <FlowSessionProvider>
        <LivePageWithSession />
      </FlowSessionProvider>
    </UIProvider>
  );
}
