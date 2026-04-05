import React, { useCallback, useEffect } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  ControlButton,
  Controls,
  useNodesState,
  useReactFlow,
  NodeChange,
  NodeTypes,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Step } from "@/app/api";
import {
  IconDiagramEmptyState,
  IconThemeDark,
  IconThemeLight,
} from "@/utils/iconRegistry";
import Node from "@/app/components/organisms/OverviewStep/Node";
import DiagramHud, {
  DiagramHudText,
} from "@/app/components/molecules/DiagramHud";
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
import { useTheme, useToggleTheme } from "@/app/store/themeStore";
import glassChromeStyles from "@/app/styles/modules/GlassChrome.module.css";
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
  const theme = useTheme();
  const toggleTheme = useToggleTheme();
  const { goalSteps, setGoalSteps } = useDiagramSelection();
  const activeGoalStepId =
    goalSteps.length > 0 ? goalSteps[goalSteps.length - 1] : null;
  const reactFlowInstance = useReactFlow();
  const viewportKey = "overview";
  const { diagramContainerRef, focusedPreviewAttribute } = useUI();
  const { previewPlan, handleStepClick, clearPreview } =
    useExecutionPlanPreview(goalSteps, setGoalSteps);

  const { visibleSteps, previewStepIds } = useStepVisibility(
    steps,
    previewPlan
  );

  const initialNodes = useNodeCalculation(
    visibleSteps,
    goalSteps,
    previewPlan,
    previewStepIds,
    handleStepClick,
    diagramContainerRef
  );

  const initialEdges = useEdgeCalculation(
    visibleSteps,
    previewStepIds,
    focusedPreviewAttribute
  );

  const { plan } = useLayoutPlan(visibleSteps, []);
  const previewStepCount = previewPlan
    ? Object.keys(previewPlan.steps).length
    : 0;
  const showPreviewHud = !!previewPlan && goalSteps.length > 0;

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
      {showPreviewHud && (
        <DiagramHud
          className={styles.previewHud}
          sections={[
            <DiagramHudText nowrap>
              {t("overview.previewLabel")}
            </DiagramHudText>,
            <DiagramHudText>
              {t("overview.previewGoals", {
                count: goalSteps.length,
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
      <ReactFlow
        nodes={nodes}
        edges={initialEdges}
        nodeTypes={nodeTypes}
        onNodesChange={handleNodesChange}
        onPaneClick={handlePaneClick}
        onNodeDragStart={handleNodeDragStart}
        nodesConnectable={false}
        nodesDraggable={true}
        elementsSelectable={false}
        nodesFocusable={false}
        onViewportChange={handleViewportChange}
        className="overview-mode-bg"
        proOptions={{ hideAttribution: true }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          className="diagram-background"
        />
        <Controls
          className={glassChromeStyles.controls}
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

const OverviewDiagramView: React.FC<OverviewDiagramViewProps> = (props) => (
  <ReactFlowProvider>
    <OverviewDiagramViewInner {...props} />
  </ReactFlowProvider>
);

export default OverviewDiagramView;
