import React, { useState } from "react";
import StepDiagram from "./StepDiagram";
import EmptyState from "../molecules/EmptyState";
import { useWorkflowWebSocket } from "../../hooks/useWorkflowWebSocket";
import WorkflowStats from "../organisms/WorkflowStats";
import { AlertCircle, Plus } from "lucide-react";
import {
  useSteps,
  useSelectedWorkflow,
  useWorkflowData,
  useExecutions,
  useResolvedAttributes,
  useWorkflowLoading,
  useIsWorkflowMode,
  useLoadSteps,
} from "../../store/workflowStore";
import ErrorBoundary from "../organisms/ErrorBoundary";
import StepEditor from "../organisms/StepEditor";
import { isValidTimestamp } from "../../utils/dates";

interface WorkflowDiagramProps {
  selectedStep?: string | null;
  onSelectStep?: (stepId: string | null) => void;
}

const WorkflowDiagram: React.FC<WorkflowDiagramProps> = ({
  selectedStep: externalSelectedStep,
  onSelectStep,
}) => {
  useWorkflowWebSocket();

  const steps = useSteps();
  const selectedWorkflow = useSelectedWorkflow();
  const workflowData = useWorkflowData();
  const executions = useExecutions();
  const resolved = useResolvedAttributes();
  const loading = useWorkflowLoading();
  const isWorkflowMode = useIsWorkflowMode();
  const loadSteps = useLoadSteps();

  const workflowNotFound = false;
  const [showCreateStepEditor, setShowCreateStepEditor] = useState(false);
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);

  const [internalSelectedStep, setInternalSelectedStep] = React.useState<
    string | null
  >(null);

  // Use external state if both selectedStep and onSelectStep are provided (controlled mode)
  // Otherwise use internal state (uncontrolled mode)
  const isControlled =
    externalSelectedStep !== undefined && onSelectStep !== undefined;
  const selectedStep = isControlled
    ? externalSelectedStep
    : internalSelectedStep;
  const setSelectedStep = isControlled ? onSelectStep : setInternalSelectedStep;

  const handleStepCreated = async () => {
    await loadSteps();
  };

  if (selectedWorkflow && workflowNotFound && !loading) {
    return (
      <div className="flex h-full items-center justify-center bg-white">
        <EmptyState
          icon={
            <AlertCircle className="text-collector-text mx-auto mb-4 h-16 w-16" />
          }
          title="Workflow Not Found"
          description={`The workflow "${selectedWorkflow}" could not be found.`}
        />
      </div>
    );
  }

  if (!steps || steps.length === 0) {
    return (
      <div className="flex h-full items-center justify-center bg-white">
        <EmptyState
          title="No Steps Registered"
          description="Register workflow steps with the Spuds engine to see the workflow diagram."
        />
      </div>
    );
  }

  const showInfoBar = !isWorkflowMode;

  return (
    <div
      className={`flex h-full flex-col ${isWorkflowMode ? "bg-neutral-label" : "bg-white"}`}
    >
      {showInfoBar ? (
        <div className="overview-header">
          <div className="overview-header__content">
            <h2 className="overview-header__title">Step Dependencies</h2>
            <div className="overview-header__stats">
              {steps.length} step{steps.length !== 1 ? "s" : ""} registered
              <button
                onClick={() => setShowCreateStepEditor(true)}
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
        workflowData && (
          <div className="workflow-header">
            <div className="workflow-header__content">
              <div className="workflow-header__left">
                <h2 className="workflow-header__title">{workflowData.id}</h2>
                <span
                  className={`status-bubble workflow-status-badge ${workflowData.status}`}
                >
                  {workflowData.status}
                </span>
                {workflowData.execution_plan?.steps && steps && (
                  <WorkflowStats
                    steps={steps}
                    executionSequence={workflowData.execution_plan.steps.map(
                      (step) => step.id
                    )}
                    resolvedAttributes={resolved}
                  />
                )}
              </div>

              <div className="workflow-header__right">
                {isValidTimestamp(workflowData.started_at) && (
                  <span>
                    Started:{" "}
                    {new Date(workflowData.started_at).toLocaleString()}
                  </span>
                )}
                {workflowData.completed_at &&
                  isValidTimestamp(workflowData.completed_at) && (
                    <span>
                      {" · "}Ended:{" "}
                      {new Date(workflowData.completed_at).toLocaleString()}
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
              <StepDiagram
                steps={steps || []}
                selectedStep={selectedStep}
                onSelectStep={setSelectedStep}
                workflowData={workflowData}
                executions={isWorkflowMode ? executions || [] : []}
                resolvedAttributes={isWorkflowMode ? resolved : []}
              />
            </ErrorBoundary>
          </div>
        )}
      </div>
      {showCreateStepEditor && (
        <StepEditor
          step={null}
          onClose={() => setShowCreateStepEditor(false)}
          onUpdate={handleStepCreated}
          diagramContainerRef={
            diagramContainerRef as React.RefObject<HTMLDivElement>
          }
        />
      )}
    </div>
  );
};

export default WorkflowDiagram;
