import React from "react";
import { Step } from "@/app/api";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { NodePositionScope } from "@/utils/nodePositioning";
import { useSpaceSelection } from "@/app/store/flowStore";
import { useUI } from "@/app/contexts/UIContext";
import { useDiagramSelection } from "@/app/contexts/DiagramSelectionContext";
import { useDiagramDetail } from "@/app/hooks/useDiagramDetail";
import { useViewportRestore } from "@/app/hooks/useViewportRestore";
import { activeGoalSet, upstreamSteps } from "@/utils/goalSets";
import { getStepsFromPlan } from "@/utils/planUtils";
import { hideTooltips } from "@/utils/tooltips";
import { useExecutionPlanPreview } from "./useExecutionPlanPreview";
import { useStepVisibility } from "./useStepVisibility";
import { useNodeCalculation } from "./useNodeCalculation";
import { useAutoLayout } from "./useAutoLayout";
import { useLayoutPlan } from "./useLayoutPlan";
import { useDiagramNodes } from "./useDiagramNodes";
import { useDiagramShortcuts } from "./useDiagramShortcuts";

const usePreviewRefresh = (
  steps: Step[],
  goalSteps: string[][],
  previewPlan: unknown
) => {
  const { updatePreviewPlan } = useUI();
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
};

const useDiagramScope = () => {
  const { spaceId } = useUI();
  const spaceSelection = useSpaceSelection();
  const selected = spaceId ? spaceSelection[spaceId] : null;
  const viewportKey = spaceId ? `space:${spaceId}` : "overview";
  const positionScope = React.useMemo<NodePositionScope>(
    () => (spaceId ? { type: "space", spaceId } : { type: "overview" }),
    [spaceId]
  );
  return { selected, viewportKey, positionScope };
};

export const useOverviewDiagramView = (steps: Step[]) => {
  const { goalSteps, setGoalSteps } = useDiagramSelection();
  const goalStepIds = React.useMemo(() => goalSteps.flat(), [goalSteps]);
  const { diagramContainerRef, focusedPreviewAttribute, hoveredGoalSet } =
    useUI();
  const previewGoalIndex = hoveredGoalSet ?? Math.max(0, goalSteps.length - 1);
  const previewGoals = React.useMemo(
    () => goalSteps[previewGoalIndex] ?? [],
    [goalSteps, previewGoalIndex]
  );
  const activeGoalStepId = activeGoalSet(goalSteps).at(-1) ?? null;
  const { selected, viewportKey, positionScope } = useDiagramScope();
  const { previewPlan, handleStepClick, clearActiveSet } =
    useExecutionPlanPreview(steps, goalSteps, setGoalSteps);

  usePreviewRefresh(steps, goalSteps, previewPlan);

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
  const { diagramDetail, visibleEdges } = useDiagramDetail(initialEdges);

  const { plan } = useLayoutPlan(visibleSteps, [], positionScope);
  const previewStepCount = focusedStepIds?.size ?? 0;
  const showPreviewHud = !!previewPlan && goalStepIds.length > 0;

  const arrangedNodes = useAutoLayout(initialNodes, initialEdges, plan);

  const { nodes, handleNodesChange } = useDiagramNodes(
    arrangedNodes,
    positionScope
  );

  const handleViewportChange = useViewportRestore(viewportKey, nodes);

  useDiagramShortcuts({
    nodes,
    activeGoalStepId,
    handleStepClick,
    diagramContainerRef,
  });

  return {
    diagramContainerRef,
    diagramDetail,
    nodes,
    visibleEdges,
    isEmpty: !visibleSteps || visibleSteps.length === 0,
    handleNodesChange,
    handleNodeDragStart: hideTooltips,
    handlePaneClick: clearActiveSet,
    handleViewportChange,
    showPreviewHud,
    previewGoalIndex,
    previewGoalCount: previewGoals.length,
    previewStepCount,
  };
};
