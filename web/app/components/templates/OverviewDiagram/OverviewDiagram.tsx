import React from "react";
import OverviewDiagramView from "@/app/components/templates/OverviewDiagramView";
import DiagramLayout from "@/app/components/templates/DiagramLayout";
import EmptyState from "@/app/components/molecules/EmptyState";
import FlowCreateForm from "@/app/components/organisms/FlowCreateForm";
import styles from "./OverviewDiagram.module.css";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { DiagramSelectionProvider } from "@/app/contexts/DiagramSelectionContext";
import { useUI } from "@/app/contexts/UIContext";
import { useFlowSession } from "@/app/contexts/FlowSessionContext";
import {
  StepEditorProvider,
  useStepEditorContext,
} from "@/app/contexts/StepEditorContext";
import { useStepEditorIntegration } from "./useStepEditorIntegration";
import { useT } from "@/app/i18n";
import { useFlowStore } from "@/app/store/flowStore";
import { Step } from "@/app/api";

interface OverviewDiagramContentProps {
  openEditor: ReturnType<typeof useStepEditorContext>["openEditor"];
}

const OverviewDiagramContent: React.FC<OverviewDiagramContentProps> = ({
  openEditor,
}) => {
  const { steps, loadSteps } = useFlowSession();
  const upsertStep = useFlowStore((state) => state.upsertStep);
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, toggleGoalStep, setGoalSteps, panelRef } = useUI();
  const t = useT();

  const applyStepUpdate = React.useCallback(
    (updatedStep: Step) => {
      upsertStep(updatedStep);
    },
    [upsertStep]
  );

  const { handleStepCreated } = useStepEditorIntegration(
    loadSteps,
    applyStepUpdate
  );
  const handleCreateStep = React.useCallback(() => {
    openEditor({
      step: null,
      diagramContainerRef,
      onUpdate: handleStepCreated,
    });
  }, [handleStepCreated, openEditor]);

  if (steps.length === 0) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          title={t("overview.noStepsTitle")}
          description={t("overview.noStepsDescription")}
        />
      </div>
    );
  }

  return (
    <DiagramLayout
      className={styles.containerOverviewMode}
      containerRef={diagramContainerRef}
    >
      <ErrorBoundary
        title={t("diagram.errorTitle")}
        description={t("diagram.errorDescription")}
      >
        <DiagramSelectionProvider
          value={{
            goalSteps,
            toggleGoalStep,
            setGoalSteps,
          }}
        >
          <div className={styles.workspace}>
            <div className={styles.diagramPane}>
              <OverviewDiagramView steps={steps} />
            </div>
            <div className={styles.panelPane} ref={panelRef}>
              <FlowCreateForm onCreateStep={handleCreateStep} />
            </div>
          </div>
        </DiagramSelectionProvider>
      </ErrorBoundary>
    </DiagramLayout>
  );
};

const OverviewDiagramInner: React.FC = () => {
  const { openEditor } = useStepEditorContext();
  return <OverviewDiagramContent openEditor={openEditor} />;
};

const OverviewDiagram: React.FC = () => (
  <StepEditorProvider>
    <OverviewDiagramInner />
  </StepEditorProvider>
);

export default OverviewDiagram;
