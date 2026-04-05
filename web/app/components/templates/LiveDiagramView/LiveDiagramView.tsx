import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  ControlButton,
  Controls,
  useReactFlow,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { FlowContext, ExecutionResult, Step } from "@/app/api";
import Node from "@/app/components/organisms/LiveStep/Node";
import {
  IconDiagramLoading,
  IconThemeDark,
  IconThemeLight,
} from "@/utils/iconRegistry";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import DiagramView from "@/app/components/molecules/DiagramView";
import { useT } from "@/app/i18n";
import { useNodeCalculation } from "./useNodeCalculation";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { STEP_LAYOUT } from "@/constants/layout";
import { useUI } from "@/app/contexts/UIContext";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
import { useStepVisibility } from "./useStepVisibility";
import { useTheme, useToggleTheme } from "@/app/store/themeStore";

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
  const theme = useTheme();
  const toggleTheme = useToggleTheme();
  const reactFlowInstance = useReactFlow();
  const viewportKey = flowData?.id || "flow";
  const { diagramContainerRef } = useUI();

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
  const handleZoomIn = useCallback(() => {
    void reactFlowInstance.zoomIn();
  }, [reactFlowInstance]);
  const handleZoomOut = useCallback(() => {
    void reactFlowInstance.zoomOut();
  }, [reactFlowInstance]);
  const handleFitView = useCallback(() => {
    void reactFlowInstance.fitView({
      padding: STEP_LAYOUT.FIT_VIEW_PADDING,
    });
  }, [reactFlowInstance]);

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
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          className="diagram-background"
        />
        <Controls
          orientation="horizontal"
          position="bottom-right"
          showInteractive={false}
          onZoomIn={handleZoomIn}
          onZoomOut={handleZoomOut}
          onFitView={handleFitView}
          style={{
            right: "1rem",
            bottom: "1rem",
          }}
        >
          <ControlButton
            onClick={toggleTheme}
            title={
              theme === "dark"
                ? t("controls.switchToLightMode")
                : t("controls.switchToDarkMode")
            }
            aria-label={
              theme === "dark"
                ? t("controls.switchToLightMode")
                : t("controls.switchToDarkMode")
            }
          >
            {theme === "dark" ? <IconThemeLight /> : <IconThemeDark />}
          </ControlButton>
        </Controls>
      </ReactFlow>
    </DiagramView>
  );
};

const LiveDiagramView: React.FC<LiveDiagramViewProps> = (props) => (
  <ReactFlowProvider>
    <LiveDiagramViewInner {...props} />
  </ReactFlowProvider>
);

export default LiveDiagramView;
