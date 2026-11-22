import React from "react";
import { ReactFlow, ReactFlowProvider, NodeTypes } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import Node from "@/app/components/organisms/LiveStep/Node";
import { IconDiagramLoading } from "@/utils/iconRegistry";
import DiagramChrome from "@/app/components/molecules/DiagramChrome";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import { useT } from "@/app/i18n";
import { MIN_DIAGRAM_ZOOM } from "@/utils/diagramDetail";
import {
  LiveDiagramViewOptions,
  useLiveDiagramView,
} from "./useLiveDiagramView";

const NODE_TYPES: NodeTypes = {
  stepNode: Node,
};

const LiveDiagramViewInner: React.FC<LiveDiagramViewOptions> = (props) => {
  const t = useT();
  const {
    diagramContainerRef,
    diagramDetail,
    nodes,
    visibleEdges,
    isEmpty,
    handleNodeDragStart,
    handleViewportChange,
  } = useLiveDiagramView(props);

  if (isEmpty) {
    return (
      <DiagramEmptyState
        icon={<IconDiagramLoading />}
        title={t("live.loadingTitle")}
        description={t("live.loadingDescription")}
      />
    );
  }

  return (
    <ReactFlow
      ref={diagramContainerRef}
      nodes={nodes}
      edges={visibleEdges}
      nodeTypes={NODE_TYPES}
      onNodeDragStart={handleNodeDragStart}
      nodesConnectable={false}
      nodesDraggable={false}
      elementsSelectable={false}
      nodesFocusable={false}
      panOnScroll={true}
      zoomOnScroll={false}
      zoomOnPinch={true}
      minZoom={MIN_DIAGRAM_ZOOM}
      onViewportChange={handleViewportChange}
      className={`flow-mode-bg diagram-detail-${diagramDetail}`}
      proOptions={{ hideAttribution: true }}
    >
      <DiagramChrome />
    </ReactFlow>
  );
};

const LiveDiagramView: React.FC<LiveDiagramViewOptions> = (props) => (
  <ReactFlowProvider>
    <LiveDiagramViewInner {...props} />
  </ReactFlowProvider>
);

export default LiveDiagramView;
