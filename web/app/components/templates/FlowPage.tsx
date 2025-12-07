"use client";

import React from "react";
import FlowDiagram from "./FlowDiagram";
import FlowSelector from "../organisms/FlowSelector";
import ErrorBoundary from "../organisms/ErrorBoundary";
import { UIProvider } from "../../contexts/UIContext";
import {
  FlowSessionProvider,
  useFlowSession,
} from "../../contexts/FlowSessionContext";

function FlowPageContent() {
  return (
    <div className="bg-neutral-bg flex h-screen flex-col">
      <ErrorBoundary
        title="Flow Selector Error"
        description="An error occurred in the flow selector. Try refreshing the page."
      >
        <FlowSelector />
      </ErrorBoundary>
      <div className="flex-1">
        <ErrorBoundary
          title="Diagram Error"
          description="An error occurred while rendering the diagram. Try selecting a different flow."
        >
          <FlowDiagram />
        </ErrorBoundary>
      </div>
    </div>
  );
}

function FlowPageWithSession() {
  const { flowError } = useFlowSession();

  if (flowError) {
    return (
      <div className="bg-neutral-bg flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-collector-text mb-4">Error: {flowError}</p>
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

  return <FlowPageContent />;
}

export default function FlowPage() {
  return (
    <UIProvider>
      <FlowSessionProvider>
        <FlowPageWithSession />
      </FlowSessionProvider>
    </UIProvider>
  );
}
