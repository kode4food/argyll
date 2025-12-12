import React, { useCallback, useEffect, useRef, useMemo } from "react";
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
  Viewport,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Step, FlowContext, ExecutionResult, AttributeRole } from "../../api";
import { Server } from "lucide-react";
import StepNode from "../organisms/StepNode";
import Legend from "../molecules/Legend";
import styles from "./StepDiagram.module.css";
import EmptyState from "../molecules/EmptyState";
import { useExecutionPlanPreview } from "../../hooks/useExecutionPlanPreview";
import { useStepVisibility } from "../../hooks/useStepVisibility";
import { useNodeCalculation } from "../../hooks/useNodeCalculation";
import { useEdgeCalculation } from "../../hooks/useEdgeCalculation";
import { useAutoLayout } from "../../hooks/useAutoLayout";
import { STEP_LAYOUT } from "@/constants/layout";
import { saveNodePositions, loadNodePositions } from "@/utils/nodePositioning";
import {
  saveViewportState,
  getViewportForKey,
} from "@/utils/viewportPersistence";
import { useUI } from "../../contexts/UIContext";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { useDiagramSelection } from "../../contexts/DiagramSelectionContext";

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

  // Generate a plan from visible steps for auto-layout in overview mode
  // Only apply auto-layout if there are no saved positions
  const overviewPlan = useMemo(() => {
    if (flowData) return null;

    const savedPositions = loadNodePositions();
    const hasSavedPositions = visibleSteps.some(
      (step) => savedPositions[step.id]
    );
    if (hasSavedPositions) return null;

    const attributes: Record<
      string,
      { providers: string[]; consumers: string[] }
    > = {};

    visibleSteps.forEach((step) => {
      Object.entries(step.attributes || {}).forEach(([attrName, attr]) => {
        if (!attributes[attrName]) {
          attributes[attrName] = { providers: [], consumers: [] };
        }

        if (attr.role === AttributeRole.Output) {
          attributes[attrName].providers.push(step.id);
        } else if (
          attr.role === AttributeRole.Required ||
          attr.role === AttributeRole.Optional
        ) {
          attributes[attrName].consumers.push(step.id);
        }
      });
    });

    return {
      attributes,
      steps: Object.fromEntries(visibleSteps.map((s) => [s.id, s])),
      goals: [],
      required: [],
    };
  }, [visibleSteps, flowData]);

  const arrangedNodes = useAutoLayout(initialNodes, initialEdges, overviewPlan);

  // Save auto-laid-out positions when they're first calculated
  useEffect(() => {
    if (!flowData && overviewPlan && arrangedNodes.length > 0) {
      saveNodePositions(arrangedNodes);
    }
  }, [arrangedNodes, flowData, overviewPlan]);

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

  const handleViewportChange = useCallback(
    (viewport: Viewport) => {
      const event = new CustomEvent("hideTooltips");
      document.dispatchEvent(event);
      saveViewportState(viewportKey, viewport);
    },
    [viewportKey]
  );

  const stepsByLevel = useMemo(() => {
    const levels = new Map<number, Array<{ id: string; y: number }>>();

    nodes.forEach((node) => {
      const level = Math.round(node.position.x / 400);
      if (!levels.has(level)) {
        levels.set(level, []);
      }
      levels.get(level)!.push({ id: node.id, y: node.position.y });
    });

    levels.forEach((steps) => {
      steps.sort((a, b) => a.y - b.y);
    });

    return levels;
  }, [nodes]);

  const findNextStep = useCallback(
    (direction: "up" | "down" | "left" | "right") => {
      if (!activeGoalStepId) {
        const firstLevel = Math.min(...Array.from(stepsByLevel.keys()));
        const stepsInLevel = stepsByLevel.get(firstLevel);
        return stepsInLevel?.[0]?.id || null;
      }

      const currentNode = nodes.find((n) => n.id === activeGoalStepId);
      if (!currentNode) return null;

      const currentLevel = Math.round(currentNode.position.x / 400);
      const currentLevelSteps = stepsByLevel.get(currentLevel) || [];
      const currentIndex = currentLevelSteps.findIndex(
        (s) => s.id === activeGoalStepId
      );

      switch (direction) {
        case "up": {
          if (currentIndex > 0) {
            return currentLevelSteps[currentIndex - 1].id;
          }
          return null;
        }
        case "down": {
          if (currentIndex < currentLevelSteps.length - 1) {
            return currentLevelSteps[currentIndex + 1].id;
          }
          return null;
        }
        case "left": {
          const prevLevel = currentLevel - 1;
          const prevLevelSteps = stepsByLevel.get(prevLevel);
          if (!prevLevelSteps || prevLevelSteps.length === 0) return null;

          const closest = prevLevelSteps.reduce((prev, curr) => {
            const prevDist = Math.abs(prev.y - currentNode.position.y);
            const currDist = Math.abs(curr.y - currentNode.position.y);
            return currDist < prevDist ? curr : prev;
          });
          return closest.id;
        }
        case "right": {
          const nextLevel = currentLevel + 1;
          const nextLevelSteps = stepsByLevel.get(nextLevel);
          if (!nextLevelSteps || nextLevelSteps.length === 0) return null;

          const closest = nextLevelSteps.reduce((prev, curr) => {
            const prevDist = Math.abs(prev.y - currentNode.position.y);
            const currDist = Math.abs(curr.y - currentNode.position.y);
            return currDist < prevDist ? curr : prev;
          });
          return closest.id;
        }
        default:
          return null;
      }
    },
    [activeGoalStepId, nodes, stepsByLevel]
  );

  useKeyboardShortcuts(
    [
      {
        key: "ArrowUp",
        description: "Navigate up within level",
        handler: () => {
          const nextStep = findNextStep("up");
          if (nextStep) handleStepClick(nextStep);
        },
      },
      {
        key: "ArrowDown",
        description: "Navigate down within level",
        handler: () => {
          const nextStep = findNextStep("down");
          if (nextStep) handleStepClick(nextStep);
        },
      },
      {
        key: "ArrowLeft",
        description: "Navigate to earlier dependency level",
        handler: () => {
          const nextStep = findNextStep("left");
          if (nextStep) handleStepClick(nextStep);
        },
      },
      {
        key: "ArrowRight",
        description: "Navigate to later dependency level",
        handler: () => {
          const nextStep = findNextStep("right");
          if (nextStep) handleStepClick(nextStep);
        },
      },
      {
        key: "Enter",
        description: "Open step editor",
        handler: () => {
          if (activeGoalStepId) {
            const event = new CustomEvent("openStepEditor", {
              detail: { stepId: activeGoalStepId },
            });
            document.dispatchEvent(event);
          }
        },
      },
      {
        key: "Escape",
        description: "Deselect step",
        handler: () => {
          if (activeGoalStepId && !flowData) {
            setGoalSteps([]);
          }
        },
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

  const shouldFitView = React.useMemo(() => {
    return !getViewportForKey(viewportKey);
  }, [viewportKey]);

  React.useEffect(() => {
    const nodesToUse = flowData ? initialNodes : arrangedNodes;

    setNodes((currentNodes) => {
      const nodeMap = new Map(currentNodes.map((n) => [n.id, n]));

      return nodesToUse.map((newNode) => {
        const existingNode = nodeMap.get(newNode.id);
        if (existingNode) {
          return {
            ...newNode,
            position: existingNode.position,
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
          className="py-12"
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
        fitView={shouldFitView}
        fitViewOptions={{ padding: STEP_LAYOUT.FIT_VIEW_PADDING }}
        onViewportChange={handleViewportChange}
        className={flowData ? "bg-diagram-flow" : "bg-neutral-bg"}
        proOptions={{ hideAttribution: true }}
      >
        <Controls showInteractive={false} className="!bottom-4 !left-4" />
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          className="opacity-30"
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
