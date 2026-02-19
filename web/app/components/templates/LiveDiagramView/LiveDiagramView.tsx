import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  BackgroundVariant,
  useReactFlow,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { FlowContext, ExecutionResult, Step } from "@/app/api";
import Node from "@/app/components/organisms/LiveStep/Node";
import Legend from "@/app/components/molecules/Legend";
import { IconDiagramLoading } from "@/utils/iconRegistry";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import DiagramView from "@/app/components/molecules/DiagramView";
import { useT } from "@/app/i18n";
import { useNodeCalculation } from "./useNodeCalculation";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { STEP_LAYOUT } from "@/constants/layout";
import { useUI } from "@/app/contexts/UIContext";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
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
  const { disableEdit, diagramContainerRef } = useUI();

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
    diagramContainerRef,
    disableEdit
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
        reactFlowInstance.fitView({
          padding: STEP_LAYOUT.FIT_VIEW_PADDING,
        });
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
  }, [reactFlowInstance, shouldFitView, nodes, markFitApplied]);

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
    <DiagramView ref={diagramContainerRef}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeDragStart={handleNodeDragStart}
        nodesConnectable={false}
        nodesDraggable={false}
        elementsSelectable={false}
        nodesFocusable={false}
        onViewportChange={handleViewportChange}
        className="flow-mode-bg"
        proOptions={{ hideAttribution: true }}
      >
        <Controls showInteractive={false} className="diagram-controls" />
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          className="diagram-background"
        />
      </ReactFlow>

      <Legend />
    </DiagramView>
  );
};

const LiveDiagramView: React.FC<LiveDiagramViewProps> = (props) => {
  return (
    <ReactFlowProvider>
      <LiveDiagramViewInner {...props} />
    </ReactFlowProvider>
  );
};

export default LiveDiagramView;
