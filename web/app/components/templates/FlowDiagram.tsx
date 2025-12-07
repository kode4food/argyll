import React from "react";
import StepDiagram from "./StepDiagram";
import EmptyState from "../molecules/EmptyState";
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
  const stepsList = steps || [];
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { selectedStep, setSelectedStep } = useUI();
  const { openEditor } = useStepEditorContext();

  const handleStepCreated = async () => {
    await loadSteps();
  };

  if (selectedFlow && flowNotFound && !loading) {
    return (
      <div className="flex h-full items-center justify-center bg-white">
        <EmptyState
          icon={
            <AlertCircle className="text-collector-text mx-auto mb-4 h-16 w-16" />
          }
          title="Flow Not Found"
          description={`The flow "${selectedFlow}" could not be found.`}
        />
      </div>
    );
  }

  if (!steps || steps.length === 0) {
    return (
      <div className="flex h-full items-center justify-center bg-white">
        <EmptyState
          title="No Steps Registered"
          description="Register flow steps with the Spuds engine to see the flow diagram."
        />
      </div>
    );
  }

  const showInfoBar = !isFlowMode;

  return (
    <div
      className={`flex h-full flex-col ${isFlowMode ? "bg-neutral-label" : "bg-white"}`}
    >
      {showInfoBar ? (
        <div className="overview-header">
          <div className="overview-header__content">
            <h2 className="overview-header__title">Step Dependencies</h2>
            <div className="overview-header__stats">
              {steps.length} step{steps.length !== 1 ? "s" : ""} registered
              <button
                onClick={() =>
                  openEditor({
                    step: null,
                    diagramContainerRef,
                    onUpdate: handleStepCreated,
                  })
                }
                className="ml-2 inline-flex items-center justify-center rounded-full bg-blue-600/20 p-1 transition-colors hover:bg-blue-600/30"
                title="Create New Step"
                aria-label="Create New Step"
              >
                <Plus className="h-4 w-4" />
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

      <div className="relative flex-1" ref={diagramContainerRef}>
        {loading ? null : (
          <div className="h-full w-full">
            <ErrorBoundary
              title="Step Diagram Error"
              description="An error occurred while rendering the step diagram."
            >
              <DiagramSelectionProvider
                value={{ selectedStep, setSelectedStep }}
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
