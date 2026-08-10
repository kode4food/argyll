import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { FlowContext, ExecutionResult, Step } from "@/app/api";
import Node from "@/app/components/organisms/LiveStep/Node";
import { IconDiagramLoading } from "@/utils/iconRegistry";
import DiagramChrome from "@/app/components/molecules/DiagramChrome";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import { useT } from "@/app/i18n";
import { useNodeCalculation } from "./useNodeCalculation";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { useUI } from "@/app/contexts/UIContext";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
import { useFitView } from "@/app/hooks/useFitView";
import { useStepVisibility } from "./useStepVisibility";

interface LiveDiagramViewProps {
  steps: Step[];
  flowData: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
}

const nodeTypes: NodeTypes = {
  stepNode: Node,
};

const LiveDiagramViewInner: React.FC<LiveDiagramViewProps> = ({
  steps = [],
  flowData,
  executions = [],
  resolvedAttributes = [],
}) => {
  const t = useT();
  const reactFlowInstance = useReactFlow();
  const viewportKey = flowData?.id || "flow";
  const { diagramContainerRef } = useUI();
  const fitView = useFitView();

  const { visibleSteps } = useStepVisibility(steps, flowData);
  const hasPlan =
    !!flowData?.plan?.steps && Object.keys(flowData.plan.steps).length > 0;
  const stepsToRender = hasPlan ? visibleSteps : [];
  const isLoadingPlan = !flowData || !hasPlan;

  const nodes = useNodeCalculation(
    stepsToRender,
    flowData,
    executions,
    resolvedAttributes,
    diagramContainerRef
  );

  const edges = useEdgeCalculation(stepsToRender, null);

  const handleNodeDragStart = useCallback(() => {
    const event = new CustomEvent("hideTooltips");
    document.dispatchEvent(event);
  }, []);
  const {
    handleViewportChange,
    shouldFitView,
    savedViewport,
    markRestored,
    markFitApplied,
  } = useDiagramViewport(viewportKey);

  useEffect(() => {
    if (savedViewport && reactFlowInstance) {
      reactFlowInstance.setViewport(savedViewport);
      requestAnimationFrame(() => markRestored());
    }
  }, [reactFlowInstance, savedViewport, markRestored]);

  useEffect(() => {
    if (!shouldFitView || !reactFlowInstance || nodes.length === 0) {
      return;
    }

    let frameA = 0;
    let frameB = 0;

    frameA = requestAnimationFrame(() => {
      frameB = requestAnimationFrame(() => {
        fitView();
        markFitApplied();
      });
    });

    return () => {
      if (frameA) {
        cancelAnimationFrame(frameA);
      }
      if (frameB) {
        cancelAnimationFrame(frameB);
      }
    };
  }, [fitView, shouldFitView, nodes, markFitApplied]);

  if (isLoadingPlan || stepsToRender.length === 0) {
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
      edges={edges}
      nodeTypes={nodeTypes}
      onNodeDragStart={handleNodeDragStart}
      nodesConnectable={false}
      nodesDraggable={false}
      elementsSelectable={false}
      nodesFocusable={false}
      panOnScroll={true}
      zoomOnScroll={false}
      zoomOnPinch={true}
      onViewportChange={handleViewportChange}
      className="flow-mode-bg"
      proOptions={{ hideAttribution: true }}
    >
      <DiagramChrome />
    </ReactFlow>
  );
};

const LiveDiagramView: React.FC<LiveDiagramViewProps> = (props) => (
  <ReactFlowProvider>
    <LiveDiagramViewInner {...props} />
  </ReactFlowProvider>
);

export default LiveDiagramView;
