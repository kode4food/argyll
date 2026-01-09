import React, { useCallback, useEffect } from "react";
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
import { Step } from "@/app/api";
import { Server } from "lucide-react";
import Node from "@/app/components/organisms/OverviewStep/Node";
import Legend from "@/app/components/molecules/Legend";
import styles from "@/app/components/templates/StepDiagram/StepDiagram.module.css";
import EmptyState from "@/app/components/molecules/EmptyState";
import { useExecutionPlanPreview } from "./useExecutionPlanPreview";
import { useStepVisibility } from "./useStepVisibility";
import { useNodeCalculation } from "./useNodeCalculation";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { useAutoLayout } from "./useAutoLayout";
import { STEP_LAYOUT } from "@/constants/layout";
import { saveNodePositions } from "@/utils/nodePositioning";
import { useUI } from "@/app/contexts/UIContext";
import { useKeyboardShortcuts } from "@/app/hooks/useKeyboardShortcuts";
import { useDiagramSelection } from "@/app/contexts/DiagramSelectionContext";
import { useKeyboardNavigation } from "./useKeyboardNavigation";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
import { useLayoutPlan } from "./useLayoutPlan";

interface OverviewDiagramViewProps {
  steps: Step[];
}

const nodeTypes: NodeTypes = {
  stepNode: Node,
};

const OverviewDiagramViewInner: React.FC<OverviewDiagramViewProps> = ({
  steps = [],
}) => {
  const { goalSteps, setGoalSteps } = useDiagramSelection();
  const activeGoalStepId =
    goalSteps.length > 0 ? goalSteps[goalSteps.length - 1] : null;
  const reactFlowInstance = useReactFlow();
  const viewportKey = "overview";
  const { disableEdit, diagramContainerRef } = useUI();
  const { previewPlan, handleStepClick, clearPreview } =
    useExecutionPlanPreview(goalSteps, setGoalSteps);

  const { visibleSteps, previewStepIds } = useStepVisibility(
    steps || [],
    previewPlan
  );

  const initialNodes = useNodeCalculation(
    visibleSteps,
    goalSteps,
    previewPlan,
    previewStepIds,
    handleStepClick,
    diagramContainerRef,
    disableEdit
  );

  const initialEdges = useEdgeCalculation(visibleSteps, previewStepIds);

  const { plan } = useLayoutPlan(visibleSteps, []);

  const arrangedNodes = useAutoLayout(initialNodes, initialEdges, plan);

  const [nodes, setNodes, onNodesChange] = useNodesState(arrangedNodes);
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
    clearPreview();
    setGoalSteps([]);
  }, [clearPreview, setGoalSteps]);

  const handleNodeDragStart = useCallback(() => {
    const event = new CustomEvent("hideTooltips");
    document.dispatchEvent(event);
  }, []);

  const {
    handleViewportChange,
    shouldFitView: fitsView,
    savedViewport,
    markRestored,
  } = useDiagramViewport(viewportKey);

  const {
    handleArrowUp,
    handleArrowDown,
    handleArrowLeft,
    handleArrowRight,
    handleEnter,
    handleEscape,
  } = useKeyboardNavigation(nodes, activeGoalStepId, handleStepClick);

  useKeyboardShortcuts([
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
  ]);

  useEffect(() => {
    if (savedViewport && reactFlowInstance) {
      reactFlowInstance.setViewport(savedViewport);
      requestAnimationFrame(() => markRestored());
    }
  }, [reactFlowInstance, savedViewport, markRestored]);

  React.useEffect(() => {
    setNodes((currentNodes) => {
      if (currentNodes.length === 0) {
        return arrangedNodes;
      }

      const nodeMap = new Map(currentNodes.map((n) => [n.id, n]));
      let hasChanges = currentNodes.length !== arrangedNodes.length;

      const nextNodes = arrangedNodes.map((newNode) => {
        const oldNode = nodeMap.get(newNode.id);
        if (oldNode) {
          if (oldNode.data === newNode.data && oldNode.type === newNode.type) {
            return oldNode;
          }
          hasChanges = true;
          return {
            ...oldNode,
            data: newNode.data,
            type: newNode.type,
          };
        }
        hasChanges = true;
        return newNode;
      });

      return hasChanges ? nextNodes : currentNodes;
    });
  }, [arrangedNodes, setNodes]);

  React.useEffect(() => {
    setEdges((currentEdges) => {
      if (currentEdges.length !== initialEdges.length) {
        return initialEdges;
      }
      for (let i = 0; i < currentEdges.length; i += 1) {
        if (
          currentEdges[i].id !== initialEdges[i].id ||
          currentEdges[i].style?.stroke !== initialEdges[i].style?.stroke
        ) {
          return initialEdges;
        }
      }
      return currentEdges;
    });
  }, [initialEdges, setEdges]);

  if (!visibleSteps || visibleSteps.length === 0) {
    return (
      <div className={styles.emptyStateWrapper}>
        <EmptyState
          icon={<Server />}
          title="No Steps to Visualize"
          description="Register steps to see their dependency relationships in diagram form."
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
        onNodesChange={handleNodesChange}
        onEdgesChange={onEdgesChange}
        onPaneClick={handlePaneClick}
        onNodeDragStart={handleNodeDragStart}
        nodesConnectable={false}
        nodesDraggable={!disableEdit}
        elementsSelectable={false}
        nodesFocusable={false}
        fitView={fitsView}
        fitViewOptions={{ padding: STEP_LAYOUT.FIT_VIEW_PADDING }}
        onViewportChange={handleViewportChange}
        className="overview-mode-bg"
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

const OverviewDiagramView: React.FC<OverviewDiagramViewProps> = (props) => {
  return (
    <ReactFlowProvider>
      <OverviewDiagramViewInner {...props} />
    </ReactFlowProvider>
  );
};

export default OverviewDiagramView;
