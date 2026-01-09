import React, { useEffect } from "react";
import OverviewDiagramView from "@/app/components/templates/OverviewDiagramView";
import EmptyState from "@/app/components/molecules/EmptyState";
import styles from "./OverviewDiagram.module.css";
import { Plus } from "lucide-react";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { DiagramSelectionProvider } from "@/app/contexts/DiagramSelectionContext";
import { useUI } from "@/app/contexts/UIContext";
import { useFlowSession } from "@/app/contexts/FlowSessionContext";
import {
  StepEditorProvider,
  useStepEditorContext,
} from "@/app/contexts/StepEditorContext";
import { useStepEditorIntegration } from "./useStepEditorIntegration";

const OverviewDiagramContent: React.FC = () => {
  const { steps, loadSteps } = useFlowSession();
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, toggleGoalStep, setGoalSteps } = useUI();
  const { openEditor } = useStepEditorContext();

  const { handleStepCreated } = useStepEditorIntegration(
    (step) =>
      openEditor({ step, diagramContainerRef, onUpdate: handleStepCreated }),
    loadSteps
  );

  useEffect(() => {
    setGoalSteps([]);
  }, [setGoalSteps]);

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

  return (
    <div className={`${styles.container} ${styles.containerOverviewMode}`}>
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
              <OverviewDiagramView steps={steps || []} />
            </DiagramSelectionProvider>
          </ErrorBoundary>
        </div>
      </div>
    </div>
  );
};

const OverviewDiagram: React.FC = () => (
  <StepEditorProvider>
    <OverviewDiagramContent />
  </StepEditorProvider>
);

export default OverviewDiagram;
