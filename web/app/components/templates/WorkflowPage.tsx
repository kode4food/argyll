"use client";

import React from "react";
import WorkflowDiagram from "./WorkflowDiagram";
import WorkflowSelector from "../organisms/WorkflowSelector";
import ErrorBoundary from "../organisms/ErrorBoundary";
import {
  useWorkflowError,
  useLoadSteps,
  useLoadWorkflows,
} from "../../store/workflowStore";
import { UIProvider, useUI } from "../../contexts/UIContext";

function WorkflowPageContent() {
  const { selectedStep, setSelectedStep } = useUI();

  return (
    <div className="bg-neutral-bg flex h-screen flex-col">
      <ErrorBoundary
        title="Workflow Selector Error"
        description="An error occurred in the workflow selector. Try refreshing the page."
      >
        <WorkflowSelector />
      </ErrorBoundary>
      <div className="flex-1">
        <ErrorBoundary
          title="Diagram Error"
          description="An error occurred while rendering the diagram. Try selecting a different workflow."
        >
          <WorkflowDiagram
            selectedStep={selectedStep}
            onSelectStep={setSelectedStep}
          />
        </ErrorBoundary>
      </div>
    </div>
  );
}

export default function WorkflowPage() {
  const error = useWorkflowError();
  const loadSteps = useLoadSteps();
  const loadWorkflows = useLoadWorkflows();

  React.useEffect(() => {
    loadSteps();
    loadWorkflows();
  }, [loadSteps, loadWorkflows]);

  if (error) {
    return (
      <div className="bg-neutral-bg flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-collector-text mb-4">Error: {error}</p>
          <button
            onClick={() => window.location.reload()}
            className="bg-info hover:bg-processor-text rounded px-4 py-2 text-white"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <UIProvider>
      <WorkflowPageContent />
    </UIProvider>
  );
}
