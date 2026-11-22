import React from "react";
import { FlowContext, ExecutionResult, Step } from "@/app/api";
import { useEdgeCalculation } from "@/app/hooks/useEdgeCalculation";
import { useUI } from "@/app/contexts/UIContext";
import { useDiagramDetail } from "@/app/hooks/useDiagramDetail";
import { useViewportRestore } from "@/app/hooks/useViewportRestore";
import { hideTooltips } from "@/utils/tooltips";
import { useNodeCalculation } from "./useNodeCalculation";
import { useStepVisibility } from "./useStepVisibility";

export interface LiveDiagramViewOptions {
  steps: Step[];
  flowData: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
}

export const useLiveDiagramView = ({
  steps = [],
  flowData,
  executions = [],
  resolvedAttributes = [],
}: LiveDiagramViewOptions) => {
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

  const goalScopeIds = React.useMemo(
    () =>
      new Set(
        nodes
          .filter((node) => !node.data.isOutsideGoalScope)
          .map((node) => node.id)
      ),
    [nodes]
  );
  const edges = useEdgeCalculation(stepsToRender, goalScopeIds);
  const { diagramDetail, visibleEdges } = useDiagramDetail(edges);
  const handleViewportChange = useViewportRestore(viewportKey, nodes);

  return {
    diagramContainerRef,
    diagramDetail,
    nodes,
    visibleEdges,
    isEmpty: isLoadingPlan || stepsToRender.length === 0,
    handleNodeDragStart: hideTooltips,
    handleViewportChange,
  };
};
