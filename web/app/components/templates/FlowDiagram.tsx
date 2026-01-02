import React from "react";
import StepDiagram from "./StepDiagram";
import EmptyState from "../molecules/EmptyState";
import styles from "./FlowDiagram.module.css";
import FlowStats from "../organisms/FlowStats";
import { AlertCircle, Plus } from "lucide-react";
import ErrorBoundary from "../organisms/ErrorBoundary";
import { isValidTimestamp } from "@/utils/dates";
import { DiagramSelectionProvider } from "../../contexts/DiagramSelectionContext";
import { useUI } from "../../contexts/UIContext";
import { useFlowSession } from "../../contexts/FlowSessionContext";
import {
  StepEditorProvider,
  useStepEditorContext,
} from "../../contexts/StepEditorContext";
import { useStepEditorIntegration } from "./FlowDiagram/useStepEditorIntegration";

const FlowDiagramContent: React.FC = () => {
  const {
    selectedFlow,
    flowData,
    executions,
    resolvedAttributes: resolved,
    loading,
    flowNotFound,
    steps,
    isFlowMode,
    loadSteps,
  } = useFlowSession();
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, toggleGoalStep, setGoalSteps } = useUI();
  const { openEditor } = useStepEditorContext();

  const { handleStepCreated } = useStepEditorIntegration(
    (step) =>
      openEditor({ step, diagramContainerRef, onUpdate: handleStepCreated }),
    loadSteps
  );

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

  if (!steps || steps.length === 0) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          title="No Steps Registered"
          description="Register flow steps with the Argyll engine to see the flow diagram."
        />
      </div>
    );
  }

  const showInfoBar = !isFlowMode;

  return (
    <div
      className={`${styles.container} ${isFlowMode ? styles.containerFlowMode : styles.containerOverviewMode}`}
    >
      {showInfoBar ? (
        <div className={styles.overviewHeader}>
          <div className={styles.overviewContent}>
            <h2 className={styles.overviewTitle}>Step Dependencies</h2>
            <div className={styles.overviewStats}>
              {steps.length} step{steps.length !== 1 ? "s" : ""} registered
              <button
                onClick={(e) => {
                  openEditor({
                    step: null,
                    diagramContainerRef,
                    onUpdate: handleStepCreated,
                  });
                  e.currentTarget.blur();
                }}
                className={styles.overviewAddStep}
                title="Create New Step"
                aria-label="Create New Step"
              >
                <Plus className={`${styles.iconMd} icon`} />
              </button>
            </div>
          </div>
        </div>
      ) : (
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
        )
      )}

      <div className={styles.diagramContainer} ref={diagramContainerRef}>
        <div className={styles.diagramContent}>
          <ErrorBoundary
            title="Step Diagram Error"
            description="An error occurred while rendering the step diagram."
          >
            <DiagramSelectionProvider
              value={{
                goalSteps,
                toggleGoalStep,
                setGoalSteps,
              }}
            >
              <StepDiagram
                key={flowData?.id ?? "overview"}
                steps={steps || []}
                flowData={flowData}
                executions={isFlowMode ? executions || [] : []}
                resolvedAttributes={isFlowMode ? resolved : []}
              />
            </DiagramSelectionProvider>
          </ErrorBoundary>
        </div>
      </div>
    </div>
  );
};

const FlowDiagram: React.FC = () => (
  <StepEditorProvider>
    <FlowDiagramContent />
  </StepEditorProvider>
);

export default FlowDiagram;
