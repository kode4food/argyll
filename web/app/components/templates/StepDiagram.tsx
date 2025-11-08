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
import { Step, WorkflowContext, ExecutionResult } from "../../api";
import { Server } from "lucide-react";
import StepNode from "../organisms/StepNode";
import EmptyState from "../molecules/EmptyState";
import { useExecutionPlanPreview } from "../../hooks/useExecutionPlanPreview";
import { useStepVisibility } from "../../hooks/useStepVisibility";
import { useNodeCalculation } from "../../hooks/useNodeCalculation";
import { useEdgeCalculation } from "../../hooks/useEdgeCalculation";
import { STEP_LAYOUT } from "@/constants/layout";
import { saveNodePositions } from "@/utils/nodePositioning";
import {
  saveViewportState,
  getViewportForKey,
} from "@/utils/viewportPersistence";
import { useUI } from "../../contexts/UIContext";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";

interface StepDiagramProps {
  steps: Step[];
  selectedStep: string | null;
  onSelectStep: (stepId: string | null) => void;
  workflowData?: WorkflowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
}

const nodeTypes: NodeTypes = {
  stepNode: StepNode,
};

const StepDiagramInner: React.FC<StepDiagramProps> = ({
  steps = [],
  selectedStep,
  onSelectStep,
  workflowData,
  executions = [],
  resolvedAttributes = [],
}) => {
  const reactFlowInstance = useReactFlow();
  const viewportKey = workflowData?.id || "overview";
  const initialViewportSet = useRef(false);
  const { disableEdit, diagramContainerRef } = useUI();
  const { previewPlan, handleStepClick, clearPreview } =
    useExecutionPlanPreview(selectedStep, onSelectStep, workflowData);

  const { visibleSteps, previewStepIds } = useStepVisibility(
    steps || [],
    workflowData,
    previewPlan
  );

  const initialNodes = useNodeCalculation(
    visibleSteps,
    selectedStep,
    workflowData,
    executions,
    previewPlan,
    previewStepIds,
    handleStepClick,
    resolvedAttributes,
    diagramContainerRef,
    disableEdit
  );

  const initialEdges = useEdgeCalculation(visibleSteps, previewStepIds);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
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
    if (!workflowData && previewPlan) {
      clearPreview();
    }
  }, [workflowData, previewPlan, clearPreview]);

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
      if (!selectedStep) {
        const firstLevel = Math.min(...Array.from(stepsByLevel.keys()));
        const stepsInLevel = stepsByLevel.get(firstLevel);
        return stepsInLevel?.[0]?.id || null;
      }

      const currentNode = nodes.find((n) => n.id === selectedStep);
      if (!currentNode) return null;

      const currentLevel = Math.round(currentNode.position.x / 400);
      const currentLevelSteps = stepsByLevel.get(currentLevel) || [];
      const currentIndex = currentLevelSteps.findIndex(
        (s) => s.id === selectedStep
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
    [selectedStep, nodes, stepsByLevel]
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
          if (selectedStep) {
            const event = new CustomEvent("openStepEditor", {
              detail: { stepId: selectedStep },
            });
            document.dispatchEvent(event);
          }
        },
      },
      {
        key: "Escape",
        description: "Deselect step",
        handler: () => {
          if (selectedStep && !workflowData) {
            onSelectStep(null);
          }
        },
      },
    ],
    !workflowData
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
    setNodes((currentNodes) => {
      const nodeMap = new Map(currentNodes.map((n) => [n.id, n]));

      return initialNodes.map((newNode) => {
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
  }, [initialNodes]);

  React.useEffect(() => {
    setEdges(initialEdges);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialEdges]);

  if (!visibleSteps || visibleSteps.length === 0) {
    return (
      <div className="overflow-hidden bg-white shadow sm:rounded-md">
        <EmptyState
          icon={<Server className="text-neutral-text mx-auto mb-4 h-12 w-12" />}
          title="No Steps to Visualize"
          description={
            workflowData?.execution_plan
              ? "Select a workflow with an execution plan to view its step diagram."
              : "Register steps to see their dependency relationships in diagram form."
          }
          className="py-12"
        />
      </div>
    );
  }

  return (
    <div className="relative h-full w-full" ref={diagramContainerRef}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={workflowData ? undefined : handleNodesChange}
        onEdgesChange={onEdgesChange}
        onPaneClick={handlePaneClick}
        onNodeDragStart={handleNodeDragStart}
        nodesConnectable={false}
        nodesDraggable={!workflowData && !disableEdit}
        elementsSelectable={false}
        nodesFocusable={false}
        fitView={shouldFitView}
        fitViewOptions={{ padding: STEP_LAYOUT.FIT_VIEW_PADDING }}
        onViewportChange={handleViewportChange}
        className={workflowData ? "bg-diagram-workflow" : "bg-neutral-bg"}
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

      <div className="absolute bottom-4 right-4 rounded-lg border bg-white p-4 text-sm shadow-lg">
        <div className="space-y-2">
          <div className="flex items-center">
            <div className="legend-box-resolver mr-2 h-4 w-4 rounded"></div>
            <span className="text-neutral-text">Resolver Steps</span>
          </div>
          <div className="flex items-center">
            <div className="legend-box-processor mr-2 h-4 w-4 rounded"></div>
            <span className="text-neutral-text">Processor Steps</span>
          </div>
          <div className="flex items-center">
            <div className="legend-box-collector mr-2 h-4 w-4 rounded"></div>
            <span className="text-neutral-text">Collector Steps</span>
          </div>
          <div className="border-neutral-border mt-3 flex items-center border-t pt-2">
            <div className="legend-line-required mr-2 h-0 w-6"></div>
            <span className="text-neutral-text">Required</span>
          </div>
          <div className="flex items-center">
            <div className="legend-line-optional mr-2 h-0 w-6"></div>
            <span className="text-neutral-text">Optional</span>
          </div>
        </div>
      </div>
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
