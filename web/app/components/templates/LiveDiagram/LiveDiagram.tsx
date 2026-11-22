import React from "react";
import LiveDiagramView from "@/app/components/templates/LiveDiagramView";
import DiagramLayout from "@/app/components/templates/DiagramLayout";
import EmptyState from "@/app/components/molecules/EmptyState";
import DiagramHud, {
  DiagramHudText,
} from "@/app/components/molecules/DiagramHud";
import styles from "./LiveDiagram.module.css";
import FlowStats from "@/app/components/organisms/FlowStats";
import { IconFlowNotFound } from "@/utils/iconRegistry";
import ErrorBoundary from "@/app/components/organisms/ErrorBoundary";
import { StepEditorProvider } from "@/app/contexts/StepEditorContext";
import { useT } from "@/app/i18n";
import { useLiveDiagram } from "./useLiveDiagram";

const LiveDiagramContent: React.FC = () => {
  const t = useT();
  const {
    flowData,
    executions,
    resolved,
    steps,
    notFoundId,
    startedAt,
    endedAt,
  } = useLiveDiagram();

  if (notFoundId) {
    return (
      <div className={styles.emptyStateContainer}>
        <EmptyState
          icon={<IconFlowNotFound />}
          iconClassName={styles.notFoundIcon}
          title={t("live.flowNotFoundTitle")}
          description={t("live.flowNotFoundDescription", {
            id: notFoundId,
          })}
        />
      </div>
    );
  }

  return (
    <DiagramLayout className={styles.containerLiveMode}>
      <ErrorBoundary
        title={t("diagram.errorTitle")}
        description={t("diagram.errorDescription")}
      >
        <div className={styles.liveCanvas}>
          {flowData && (
            <DiagramHud
              className={styles.flowHud}
              sections={[
                <DiagramHudText nowrap>{flowData.id}</DiagramHudText>,
                <>
                  <span
                    className={`status-bubble ${styles.flowStatusBadge} ${styles[flowData.status]}`}
                  >
                    {flowData.status}
                  </span>
                  {flowData.plan?.steps && (
                    <FlowStats
                      steps={steps}
                      executionSequence={Object.keys(flowData.plan.steps)}
                      resolvedAttributes={resolved}
                    />
                  )}
                </>,
                <>
                  {startedAt && (
                    <DiagramHudText>
                      {t("common.started")}: {startedAt}
                    </DiagramHudText>
                  )}
                  {endedAt && (
                    <DiagramHudText>
                      {" · "}
                      {t("common.ended")}: {endedAt}
                    </DiagramHudText>
                  )}
                </>,
              ]}
            />
          )}

          <LiveDiagramView
            steps={steps}
            flowData={flowData}
            executions={executions}
            resolvedAttributes={resolved}
          />
        </div>
      </ErrorBoundary>
    </DiagramLayout>
  );
};

const LiveDiagram: React.FC = () => (
  <StepEditorProvider>
    <LiveDiagramContent />
  </StepEditorProvider>
);

export default LiveDiagram;
