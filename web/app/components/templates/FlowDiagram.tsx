import React from "react";
import StepDiagram from "./StepDiagram";
import EmptyState from "../molecules/EmptyState";
import styles from "./FlowDiagram.module.css";
import { useFlowWebSocket } from "../../hooks/useFlowWebSocket";
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

const FlowDiagramContent: React.FC = () => {
  useFlowWebSocket();

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

  const handleStepCreated = async () => {
    await loadSteps();
  };

  if (selectedFlow && flowNotFound && !loading) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          icon={<AlertCircle />}
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
        <div className="overview-header">
          <div className="overview-header__content">
            <h2 className="overview-header__title">Step Dependencies</h2>
            <div className="overview-header__stats">
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
                className="overview-header__add-step"
                title="Create New Step"
                aria-label="Create New Step"
              >
                <Plus className="icon" />
              </button>
            </div>
          </div>
        </div>
      ) : (
        flowData && (
          <div className="flow-header">
            <div className="flow-header__content">
              <div className="flow-header__left">
                <h2 className="flow-header__title">{flowData.id}</h2>
                <span
                  className={`status-bubble flow-status-badge ${flowData.status}`}
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

              <div className="flow-header__right">
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
        {loading ? null : (
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
                  steps={steps || []}
                  flowData={flowData}
                  executions={isFlowMode ? executions || [] : []}
                  resolvedAttributes={isFlowMode ? resolved : []}
                />
              </DiagramSelectionProvider>
            </ErrorBoundary>
          </div>
        )}
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
