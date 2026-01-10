import React, { useEffect } from "react";
import OverviewDiagramView from "@/app/components/templates/OverviewDiagramView";
import DiagramLayout from "@/app/components/templates/DiagramLayout";
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
import { useT } from "@/app/i18n";

const OverviewDiagramContent: React.FC = () => {
  const { steps, loadSteps } = useFlowSession();
  const diagramContainerRef = React.useRef<HTMLDivElement>(null);
  const { goalSteps, toggleGoalStep, setGoalSteps } = useUI();
  const { openEditor } = useStepEditorContext();
  const t = useT();

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
      header={
        <div className={styles.overviewHeader}>
          <div className={styles.overviewContent}>
            <h2 className={styles.overviewTitle}>{t("overview.title")}</h2>
            <div className={styles.overviewStats}>
              {t("overview.stepsRegistered", {
                count: steps.length,
              })}
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
                title={t("overview.addStep")}
                aria-label={t("overview.addStep")}
              >
                <Plus className={`${styles.iconMd} icon`} />
              </button>
            </div>
          </div>
        </div>
      }
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
          <OverviewDiagramView steps={steps || []} />
        </DiagramSelectionProvider>
      </ErrorBoundary>
    </DiagramLayout>
  );
};

const OverviewDiagram: React.FC = () => (
  <StepEditorProvider>
    <OverviewDiagramContent />
  </StepEditorProvider>
);

export default OverviewDiagram;
