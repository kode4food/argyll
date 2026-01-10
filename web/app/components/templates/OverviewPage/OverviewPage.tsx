import React from "react";
import OverviewDiagram from "@/app/components/templates/OverviewDiagram";
import FlowSelector from "@/app/components/organisms/FlowSelector";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { UIProvider } from "@/app/contexts/UIContext";
import {
  FlowSessionProvider,
  useFlowSession,
} from "@/app/contexts/FlowSessionContext";
import styles from "./OverviewPage.module.css";
import { useT } from "@/app/i18n";

function OverviewPageContent() {
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
          <OverviewDiagram />
        </ErrorBoundary>
      </div>
    </div>
  );
}

function OverviewPageWithSession() {
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

  return <OverviewPageContent />;
}

export default function OverviewPage() {
  return (
    <UIProvider>
      <FlowSessionProvider>
        <OverviewPageWithSession />
      </FlowSessionProvider>
    </UIProvider>
  );
}
