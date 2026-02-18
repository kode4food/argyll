import React, { useEffect } from "react";
import LiveDiagramView from "@/app/components/templates/LiveDiagramView";
import DiagramLayout from "@/app/components/templates/DiagramLayout";
import EmptyState from "@/app/components/molecules/EmptyState";
import styles from "./LiveDiagram.module.css";
import FlowStats from "@/app/components/organisms/FlowStats";
import { IconFlowNotFound } from "@/utils/iconRegistry";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { isValidTimestamp } from "@/utils/dates";
import { useUI } from "@/app/contexts/UIContext";
import { useFlowSession } from "@/app/contexts/FlowSessionContext";
import { StepEditorProvider } from "@/app/contexts/StepEditorContext";
import { useT } from "@/app/i18n";

const LiveDiagramContent: React.FC = () => {
  const {
    selectedFlow,
    flowData,
    executions,
    resolvedAttributes: resolved,
    loading,
    flowNotFound,
    steps,
  } = useFlowSession();
  const { clearPreviewPlan, setGoalSteps } = useUI();
  const t = useT();

  useEffect(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [clearPreviewPlan, setGoalSteps]);

  if (selectedFlow && flowNotFound && !loading) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          icon={<IconFlowNotFound />}
          iconClassName={styles.notFoundIcon}
          title={t("live.flowNotFoundTitle")}
          description={t("live.flowNotFoundDescription", {
            id: selectedFlow,
          })}
        />
      </div>
    );
  }

  return (
    <DiagramLayout
      className={styles.containerLiveMode}
      header={
        flowData && (
          <div className={styles.flowHeader}>
            <div className={styles.flowContent}>
              <div className={styles.flowLeft}>
                <h2 className={styles.flowTitle}>{flowData.id}</h2>
                <span
                  className={`status-bubble ${styles.flowStatusBadge} ${styles[flowData.status]}`}
                >
                  {flowData.status}
                </span>
                {flowData.plan?.steps && (
                  <FlowStats
                    steps={steps}
                    executionSequence={Object.keys(flowData.plan.steps)}
                    resolvedAttributes={resolved}
                  />
                )}
              </div>

              <div className={styles.flowRight}>
                {isValidTimestamp(flowData.started_at) && (
                  <span>
                    {t("common.started")}:{" "}
                    {new Date(flowData.started_at).toLocaleString()}
                  </span>
                )}
                {flowData.completed_at &&
                  isValidTimestamp(flowData.completed_at) && (
                    <span>
                      {" · "}
                      {t("common.ended")}:{" "}
                      {new Date(flowData.completed_at).toLocaleString()}
                    </span>
                  )}
              </div>
            </div>
          </div>
        )
      }
    >
      <ErrorBoundary
        title={t("diagram.errorTitle")}
        description={t("diagram.errorDescription")}
      >
        <LiveDiagramView
          steps={steps}
          flowData={flowData}
          executions={executions}
          resolvedAttributes={resolved}
        />
      </ErrorBoundary>
    </DiagramLayout>
  );
};

const LiveDiagram: React.FC = () => (
  <StepEditorProvider>
    <LiveDiagramContent />
  </StepEditorProvider>
);

export default LiveDiagram;
