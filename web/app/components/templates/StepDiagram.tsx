import React, { useCallback, useEffect, useRef } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  useReactFlow,
  NodeChange,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Step, FlowContext, ExecutionResult } from "../../api";
import { Server } from "lucide-react";
import StepNode from "../organisms/StepNode";
import Legend from "../molecules/Legend";
import styles from "./StepDiagram.module.css";
import EmptyState from "../molecules/EmptyState";
import { useExecutionPlanPreview } from "./StepDiagram/useExecutionPlanPreview";
import { useStepVisibility } from "./StepDiagram/useStepVisibility";
import { useNodeCalculation } from "./StepDiagram/useNodeCalculation";
import { useEdgeCalculation } from "./StepDiagram/useEdgeCalculation";
import { useAutoLayout } from "./StepDiagram/useAutoLayout";
import { STEP_LAYOUT } from "@/constants/layout";
import { saveNodePositions } from "./StepDiagram/nodePositioning";
import { getViewportForKey } from "./StepDiagram/viewportPersistence";
import { useUI } from "../../contexts/UIContext";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { useDiagramSelection } from "../../contexts/DiagramSelectionContext";
import { useDiagramKeyboardNavigation } from "./StepDiagram/useDiagramKeyboardNavigation";
import { useDiagramViewport } from "./StepDiagram/useDiagramViewport";
import { useOverviewAutoLayout } from "./StepDiagram/useOverviewAutoLayout";

interface StepDiagramProps {
  steps: Step[];
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
}

const nodeTypes: NodeTypes = {
  stepNode: StepNode,
};

const StepDiagramInner: React.FC<StepDiagramProps> = ({
  steps = [],
  flowData,
  executions = [],
  resolvedAttributes = [],
}) => {
  const { goalSteps, setGoalSteps } = useDiagramSelection();
  const activeGoalStepId =
    goalSteps.length > 0 ? goalSteps[goalSteps.length - 1] : null;
  const reactFlowInstance = useReactFlow();
  const viewportKey = flowData?.id || "overview";
  const initialViewportSet = useRef(false);
  const { disableEdit, diagramContainerRef } = useUI();
  const { previewPlan, handleStepClick, clearPreview } =
    useExecutionPlanPreview(goalSteps, setGoalSteps, flowData);

  const { visibleSteps, previewStepIds } = useStepVisibility(
    steps || [],
    flowData,
    previewPlan
  );

  const initialNodes = useNodeCalculation(
    visibleSteps,
    goalSteps,
    flowData,
    executions,
    previewPlan,
    previewStepIds,
    handleStepClick,
    resolvedAttributes,
    diagramContainerRef,
    disableEdit
  );

  const initialEdges = useEdgeCalculation(visibleSteps, previewStepIds);

  const { overviewPlan } = useOverviewAutoLayout(
    visibleSteps,
    flowData || null,
    []
  );

  const arrangedNodes = useAutoLayout(initialNodes, initialEdges, overviewPlan);

  const [nodes, setNodes, onNodesChange] = useNodesState(
    flowData ? initialNodes : arrangedNodes
  );
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes);

      const positionChanges = changes.filter(
        (change) => change.type === "position" && change.dragging === false
      );
      if (positionChanges.length > 0) {
        setTimeout(() => {
          setNodes((currentNodes) => {
            saveNodePositions(currentNodes);
            return currentNodes;
          });
        }, 0);
      }
    },
    [onNodesChange, setNodes]
  );

  const handlePaneClick = useCallback(() => {
    if (flowData) return;
    clearPreview();
    setGoalSteps([]);
  }, [flowData, clearPreview, setGoalSteps]);

  const handleNodeDragStart = useCallback(() => {
    const event = new CustomEvent("hideTooltips");
    document.dispatchEvent(event);
  }, []);

  const { handleViewportChange, shouldFitView: fitsView } =
    useDiagramViewport(viewportKey);

  const {
    handleArrowUp,
    handleArrowDown,
    handleArrowLeft,
    handleArrowRight,
    handleEnter,
    handleEscape,
  } = useDiagramKeyboardNavigation(nodes, activeGoalStepId, handleStepClick);

  useKeyboardShortcuts(
    [
      {
        key: "ArrowUp",
        description: "Navigate up within level",
        handler: handleArrowUp,
      },
      {
        key: "ArrowDown",
        description: "Navigate down within level",
        handler: handleArrowDown,
      },
      {
        key: "ArrowLeft",
        description: "Navigate to earlier dependency level",
        handler: handleArrowLeft,
      },
      {
        key: "ArrowRight",
        description: "Navigate to later dependency level",
        handler: handleArrowRight,
      },
      {
        key: "Enter",
        description: "Open step editor",
        handler: handleEnter,
      },
      {
        key: "Escape",
        description: "Deselect step",
        handler: handleEscape,
      },
    ],
    !flowData
  );

  useEffect(() => {
    if (!initialViewportSet.current && reactFlowInstance) {
      const savedViewport = getViewportForKey(viewportKey);
      if (savedViewport) {
        reactFlowInstance.setViewport(savedViewport);
        initialViewportSet.current = true;
      }
    }
  }, [reactFlowInstance, viewportKey]);

  useEffect(() => {
    initialViewportSet.current = false;
  }, [viewportKey]);

  React.useEffect(() => {
    const nodesToUse = flowData ? initialNodes : arrangedNodes;

    setNodes((currentNodes) => {
      const nodeMap = new Map(currentNodes.map((n) => [n.id, n]));

      return nodesToUse.map((newNode) => {
        const existingNode = nodeMap.get(newNode.id);
        if (existingNode) {
          return {
            ...existingNode,
            data: newNode.data,
            type: newNode.type,
          };
        }
        return newNode;
      });
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialNodes, flowData]);

  React.useEffect(() => {
    setEdges(initialEdges);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialEdges]);

  if (!visibleSteps || visibleSteps.length === 0) {
    return (
      <div className={styles.emptyStateWrapper}>
        <EmptyState
          icon={<Server />}
          title="No Steps to Visualize"
          description={
            flowData?.plan
              ? "Select a flow with an execution plan to view its step diagram."
              : "Register steps to see their dependency relationships in diagram form."
          }
          className={styles.emptyStatePadding}
        />
      </div>
    );
  }

  return (
    <div className={styles.diagramWrapper} ref={diagramContainerRef}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={flowData ? undefined : handleNodesChange}
        onEdgesChange={onEdgesChange}
        onPaneClick={handlePaneClick}
        onNodeDragStart={handleNodeDragStart}
        nodesConnectable={false}
        nodesDraggable={!flowData && !disableEdit}
        elementsSelectable={false}
        nodesFocusable={false}
        fitView={fitsView}
        fitViewOptions={{ padding: STEP_LAYOUT.FIT_VIEW_PADDING }}
        onViewportChange={handleViewportChange}
        className={flowData ? "flow-mode-bg" : "overview-mode-bg"}
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
    </div>
  );
};

const StepDiagram: React.FC<StepDiagramProps> = (props) => {
  return (
    <ReactFlowProvider>
      <StepDiagramInner {...props} />
    </ReactFlowProvider>
  );
};

export default StepDiagram;
