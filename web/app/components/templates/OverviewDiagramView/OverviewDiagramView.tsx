import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useReactFlow,
  NodeChange,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Step } from "@/app/api";
import { IconDiagramEmptyState } from "@/utils/iconRegistry";
import Node from "@/app/components/organisms/OverviewStep/Node";
import Legend from "@/app/components/molecules/Legend";
import DiagramEmptyState from "@/app/components/molecules/DiagramEmptyState";
import DiagramView from "@/app/components/molecules/DiagramView";
import { useT } from "@/app/i18n";
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
  const t = useT();
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
      description: t("keyboardShortcuts.navigateUp"),
      handler: handleArrowUp,
    },
    {
      key: "ArrowDown",
      description: t("keyboardShortcuts.navigateDown"),
      handler: handleArrowDown,
    },
    {
      key: "ArrowLeft",
      description: t("keyboardShortcuts.navigateLeft"),
      handler: handleArrowLeft,
    },
    {
      key: "ArrowRight",
      description: t("keyboardShortcuts.navigateRight"),
      handler: handleArrowRight,
    },
    {
      key: "Enter",
      description: t("keyboardShortcuts.openEditor"),
      handler: handleEnter,
    },
    {
      key: "Escape",
      description: t("keyboardShortcuts.deselectStep"),
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

  if (!visibleSteps || visibleSteps.length === 0) {
    return (
      <DiagramEmptyState
        icon={<IconDiagramEmptyState />}
        title={t("overview.noVisibleTitle")}
        description={t("overview.noVisibleDescription")}
      />
    );
  }

  return (
    <DiagramView ref={diagramContainerRef}>
      <ReactFlow
        nodes={nodes}
        edges={initialEdges}
        nodeTypes={nodeTypes}
        onNodesChange={handleNodesChange}
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
    </DiagramView>
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
