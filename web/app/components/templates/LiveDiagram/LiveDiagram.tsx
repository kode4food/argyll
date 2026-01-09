import React, { useEffect } from "react";
import LiveDiagramView from "@/app/components/templates/LiveDiagramView";
import EmptyState from "@/app/components/molecules/EmptyState";
import styles from "@/app/components/templates/OverviewDiagram/OverviewDiagram.module.css";
import FlowStats from "@/app/components/organisms/FlowStats";
import { AlertCircle } from "lucide-react";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { isValidTimestamp } from "@/utils/dates";
import { useUI } from "@/app/contexts/UIContext";
import { useFlowSession } from "@/app/contexts/FlowSessionContext";
import { StepEditorProvider } from "@/app/contexts/StepEditorContext";

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

  useEffect(() => {
    clearPreviewPlan();
    setGoalSteps([]);
  }, [clearPreviewPlan, setGoalSteps]);

  if (selectedFlow && flowNotFound && !loading) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          icon={<AlertCircle />}
          iconClassName={styles.notFoundIcon}
          title="Flow Not Found"
          description={`The flow "${selectedFlow}" could not be found.`}
        />
      </div>
    );
  }

  return (
    <div className={`${styles.container} ${styles.containerFlowMode}`}>
      {flowData && (
        <div className={styles.flowHeader}>
          <div className={styles.flowContent}>
            <div className={styles.flowLeft}>
              <h2 className={styles.flowTitle}>{flowData.id}</h2>
              <span
                className={`status-bubble ${styles.flowStatusBadge} ${styles[flowData.status]}`}
              >
                {flowData.status}
              </span>
              {flowData.plan?.steps && steps && (
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
                  Started: {new Date(flowData.started_at).toLocaleString()}
                </span>
              )}
              {flowData.completed_at &&
                isValidTimestamp(flowData.completed_at) && (
                  <span>
                    {" · "}Ended:{" "}
                    {new Date(flowData.completed_at).toLocaleString()}
                  </span>
                )}
            </div>
          </div>
        </div>
      )}

      <div className={styles.diagramContainer}>
        <div className={styles.diagramContent}>
          <ErrorBoundary
            title="Step Diagram Error"
            description="An error occurred while rendering the step diagram."
          >
            <LiveDiagramView
              steps={steps || []}
              flowData={flowData}
              executions={executions || []}
              resolvedAttributes={resolved}
            />
          </ErrorBoundary>
        </div>
      </div>
    </div>
  );
};

const LiveDiagram: React.FC = () => (
  <StepEditorProvider>
    <LiveDiagramContent />
  </StepEditorProvider>
);

export default LiveDiagram;
