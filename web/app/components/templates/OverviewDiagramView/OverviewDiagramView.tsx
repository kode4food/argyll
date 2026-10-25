import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  useNodesState,
  useReactFlow,
  useViewport,
  NodeChange,
  NodeTypes,
} from "@xyflow/react";
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
import { useExecutionPlanPreview } from "./useExecutionPlanPreview";
import { useStepVisibility } from "./useStepVisibility";
import { useNodeCalculation } from "./useNodeCalculation";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { useAutoLayout } from "./useAutoLayout";
import { NodePositionScope, saveNodePositions } from "@/utils/nodePositioning";
import { useSpaceSelection } from "@/app/store/flowStore";
import { useUI } from "@/app/contexts/UIContext";
import { useKeyboardShortcuts } from "@/app/hooks/useKeyboardShortcuts";
import { useDiagramSelection } from "@/app/contexts/DiagramSelectionContext";
import { useKeyboardNavigation } from "./useKeyboardNavigation";
import { useDiagramViewport } from "@/app/hooks/useDiagramViewport";
import { useFitView } from "@/app/hooks/useFitView";
import { useFullDetailMinZoom } from "@/app/hooks/useFullDetailMinZoom";
import useFocusWithin from "@/app/hooks/useFocusWithin";
import { useLayoutPlan } from "./useLayoutPlan";
import { activeGoalSet, upstreamSteps } from "@/utils/goalSets";
import { getStepsFromPlan } from "@/utils/planUtils";
import {
  MIN_DIAGRAM_ZOOM,
  diagramDetailForZoom,
  edgesForDiagramDetail,
} from "@/utils/diagramDetail";
import styles from "./OverviewDiagramView.module.css";

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
  const goalStepIds = React.useMemo(() => goalSteps.flat(), [goalSteps]);
  const reactFlowInstance = useReactFlow();
  const { zoom } = useViewport();
  const fullDetailMinZoom = useFullDetailMinZoom();
  const diagramDetail = diagramDetailForZoom(zoom, fullDetailMinZoom);
  const {
    diagramContainerRef,
    focusedPreviewAttribute,
    updatePreviewPlan,
    spaceId,
    hoveredGoalSet,
  } = useUI();
  const previewGoalIndex = hoveredGoalSet ?? Math.max(0, goalSteps.length - 1);
  const previewGoals = React.useMemo(
    () => goalSteps[previewGoalIndex] ?? [],
    [goalSteps, previewGoalIndex]
  );
  const activeGoalStepId = activeGoalSet(goalSteps).at(-1) ?? null;
  const spaceSelection = useSpaceSelection();
  const selected = spaceId ? spaceSelection[spaceId] : null;
  const viewportKey = spaceId ? `space:${spaceId}` : "overview";
  const positionScope = React.useMemo<NodePositionScope>(
    () => (spaceId ? { type: "space", spaceId } : { type: "overview" }),
    [spaceId]
  );
  const fitView = useFitView();
  const { previewPlan, handleStepClick } = useExecutionPlanPreview(
    steps,
    goalSteps,
    setGoalSteps
  );

  const goalStepsRef = React.useRef(goalSteps);
  const previewPlanRef = React.useRef(previewPlan);
  React.useEffect(() => {
    goalStepsRef.current = goalSteps;
    previewPlanRef.current = previewPlan;
  });

  React.useEffect(() => {
    if (goalStepsRef.current.flat().length > 0 && previewPlanRef.current) {
      void updatePreviewPlan(goalStepsRef.current, {});
    }
  }, [steps, updatePreviewPlan]);

  const { visibleSteps, previewStepIds } = useStepVisibility(
    steps,
    previewPlan,
    selected
  );

  const focusedStepIds = React.useMemo(() => {
    if (!previewPlan) return previewStepIds;
    const plannedSteps = getStepsFromPlan(previewPlan);
    return new Set([
      ...previewGoals,
      ...upstreamSteps(plannedSteps, previewGoals),
    ]);
  }, [previewGoals, previewPlan, previewStepIds]);

  const initialNodes = useNodeCalculation(
    visibleSteps,
    previewGoals,
    previewPlan,
    focusedStepIds,
    handleStepClick,
    diagramContainerRef,
    positionScope
  );

  const initialEdges = useEdgeCalculation(
    visibleSteps,
    focusedStepIds,
    focusedPreviewAttribute
  );
  const visibleEdges = React.useMemo(
    () => edgesForDiagramDetail(initialEdges, diagramDetail, zoom),
    [initialEdges, diagramDetail, zoom]
  );

  const { plan } = useLayoutPlan(visibleSteps, [], positionScope);
  const previewStepCount = focusedStepIds?.size ?? 0;
  const showPreviewHud = !!previewPlan && goalStepIds.length > 0;

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
            saveNodePositions(currentNodes, positionScope);
            return currentNodes;
          });
        }, 0);
      }
    },
    [onNodesChange, setNodes, positionScope]
  );

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

  const {
    handleArrowUp,
    handleArrowDown,
    handleArrowLeft,
    handleArrowRight,
    handleEnter,
    handleEscape,
  } = useKeyboardNavigation(nodes, activeGoalStepId, handleStepClick);

  const hasDiagramFocus = useFocusWithin(diagramContainerRef);

  useKeyboardShortcuts(
    [
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
    ],
    hasDiagramFocus
  );

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
    <ReactFlow
      ref={diagramContainerRef}
      nodes={nodes}
      edges={visibleEdges}
      nodeTypes={nodeTypes}
      onNodesChange={handleNodesChange}
      onNodeDragStart={handleNodeDragStart}
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
                count: previewGoals.length,
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
