import React from "react";
import { ReactFlow, ReactFlowProvider, NodeTypes } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Step } from "@/app/api";
import { IconDiagramEmptyState } from "@/utils/iconRegistry";
import Node from "@/app/components/organisms/OverviewStep/Node";
import DiagramChrome from "@/app/components/molecules/DiagramChrome";
import DiagramHud, {
  DiagramHudText,
} from "@/app/components/molecules/DiagramHud";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import { useT } from "@/app/i18n";
import { useUI } from "@/app/contexts/UIContext";
import { MIN_DIAGRAM_ZOOM } from "@/utils/diagramDetail";
import { useOverviewDiagramView } from "./useOverviewDiagramView";
import styles from "./OverviewDiagramView.module.css";

interface OverviewDiagramViewProps {
  steps: Step[];
}

const NODE_TYPES: NodeTypes = {
  stepNode: Node,
};

const OverviewDiagramViewInner: React.FC<OverviewDiagramViewProps> = ({
  steps = [],
}) => {
  const t = useT();
  const {
    diagramContainerRef,
    diagramDetail,
    nodes,
    visibleEdges,
    isEmpty,
    handleNodesChange,
    handleNodeDragStart,
    handlePaneClick,
    handleViewportChange,
    showPreviewHud,
    previewGoalIndex,
    previewGoalCount,
    previewStepCount,
  } = useOverviewDiagramView(steps);

  if (isEmpty) {
    return (
      <DiagramEmptyState
        icon={<IconDiagramEmptyState />}
        title={t("overview.noVisibleTitle")}
        description={t("overview.noVisibleDescription")}
      />
    );
  }

  return (
    <ReactFlow
      ref={diagramContainerRef}
      nodes={nodes}
      edges={visibleEdges}
      nodeTypes={NODE_TYPES}
      onNodesChange={handleNodesChange}
      onNodeDragStart={handleNodeDragStart}
      onPaneClick={handlePaneClick}
      nodesConnectable={false}
      nodesDraggable={true}
      elementsSelectable={false}
      nodesFocusable={false}
      panOnScroll={true}
      zoomOnScroll={false}
      zoomOnPinch={true}
      minZoom={MIN_DIAGRAM_ZOOM}
      onViewportChange={handleViewportChange}
      className={`overview-mode-bg diagram-detail-${diagramDetail}`}
      proOptions={{ hideAttribution: true }}
    >
      {showPreviewHud && (
        <DiagramHud
          className={styles.previewHud}
          sections={[
            <DiagramHudText nowrap>
              {t("overview.previewLabel")}
              {" · "}
              {previewGoalIndex === 0
                ? t("goals.primary")
                : t("goals.fallback", { count: previewGoalIndex })}
            </DiagramHudText>,
            <DiagramHudText>
              {t("overview.previewGoals", {
                count: previewGoalCount,
              })}
            </DiagramHudText>,
            <DiagramHudText>
              {t("overview.previewSteps", {
                count: previewStepCount,
              })}
            </DiagramHudText>,
          ]}
        />
      )}
      <DiagramChrome />
    </ReactFlow>
  );
};

const OverviewDiagramView: React.FC<OverviewDiagramViewProps> = (props) => {
  const { spaceId } = useUI();
  return (
    <ReactFlowProvider key={spaceId ?? "overview"}>
      <OverviewDiagramViewInner {...props} />
    </ReactFlowProvider>
  );
};

export default OverviewDiagramView;
