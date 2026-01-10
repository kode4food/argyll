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
import { useT } from "@/app/i18n";

function LivePageContent() {
  const t = useT();

  return (
    <div className={styles.page}>
      <ErrorBoundary
        title={t("diagram.selectorErrorTitle")}
        description={t("diagram.selectorErrorDescription")}
      >
        <FlowSelector />
      </ErrorBoundary>
      <div className={styles.mainContent}>
        <ErrorBoundary
          title={t("diagram.diagramErrorTitle")}
          description={t("diagram.diagramErrorDescription")}
        >
          <LiveDiagram />
        </ErrorBoundary>
      </div>
    </div>
  );
}

function LivePageWithSession() {
  const { flowError } = useFlowSession();
  const t = useT();

  if (flowError) {
    return (
      <div className={styles.errorPage}>
        <div className={styles.errorContent}>
          <p className={styles.errorMessage}>
            {t("common.errorMessage", { message: flowError })}
          </p>
          <button
            onClick={() => window.location.reload()}
            className={styles.retryButton}
          >
            {t("common.retry")}
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
