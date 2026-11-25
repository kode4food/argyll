import React, { useMemo } from "react";
import { Node } from "@xyflow/react";
import { toGoalSets, Step, FlowContext, ExecutionResult } from "@/app/api";
import { STEP_LAYOUT } from "@/constants/layout";
import { loadNodePositions, NodePositions } from "@/utils/nodePositioning";
import {
  buildOutputProducerMap,
  buildStepGraph,
  calculateStepLevels,
} from "@/utils/stepDependencyGraph";
import { calculateWidgetHeightFromAttributes } from "@/utils/stepLayout";
import { currentGoalScope } from "@/utils/goalSets";

export interface NodeCalculationOptions {
  visibleSteps: Step[];
  flowData?: FlowContext | null;
  executions?: ExecutionResult[];
  resolvedAttributes?: string[];
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
  spaceId?: string | null;
}

export const useNodeCalculation = ({
  visibleSteps,
  flowData,
  executions,
  resolvedAttributes,
  diagramContainerRef,
  spaceId,
}: NodeCalculationOptions) => {
  return useMemo(() => {
    const savedPositions = flowData?.id
      ? flowPositions(flowData.id, spaceId)
      : {};
    const activeStepIDs = new Set(visibleSteps.map((step) => step.id));
    const producerMap = buildOutputProducerMap(visibleSteps);
    const { dependencies, stepsWithDependencies } = buildStepGraph(
      visibleSteps,
      producerMap,
      activeStepIDs
    );
    const levels = calculateStepLevels(visibleSteps, dependencies);
    const goalSets = toGoalSets(flowData?.plan?.goals);
    const goalScope = currentGoalScope(
      visibleSteps,
      goalSets,
      executions ?? []
    );
    const startingPoints = new Set<string>();

    activeStepIDs.forEach((stepID) => {
      if (!stepsWithDependencies.has(stepID)) {
        startingPoints.add(stepID);
      }
    });

    const levelGroups = new Map<number, string[]>();
    levels.forEach((level, stepID) => {
      if (!levelGroups.has(level)) levelGroups.set(level, []);
      levelGroups.get(level)?.push(stepID);
    });

    return visibleSteps.map((step) => {
      let position;
      if (savedPositions[step.id]) {
        position = savedPositions[step.id];
      } else {
        const level = levels.get(step.id) || 0;
        const levelSteps = levelGroups.get(level) || [];
        const indexInLevel = levelSteps.indexOf(step.id);
        const levelSize = levelSteps.length;
        const widgetHeight = calculateWidgetHeightFromAttributes(
          step.attributes
        );

        const col = level;
        const row = indexInLevel - (levelSize - 1) / 2;

        position = {
          x: col * STEP_LAYOUT.HORIZONTAL_SPACING,
          y:
            row * (widgetHeight + STEP_LAYOUT.VERTICAL_SPACING) +
            STEP_LAYOUT.VERTICAL_OFFSET,
        };
      }

      const node: Node = {
        id: step.id,
        position,
        data: {
          step,
          selected: false,
          flowData,
          executions,
          resolvedAttributes,
          isGoalStep: goalSets.flat().includes(step.id),
          isStartingPoint: startingPoints.has(step.id),
          isOutsideGoalScope: !goalScope.has(step.id),
          diagramContainerRef,
        },
        type: "stepNode",
      };

      return node;
    });
  }, [
    visibleSteps,
    flowData,
    executions,
    resolvedAttributes,
    diagramContainerRef,
    spaceId,
  ]);
};

// A Flow started outside the UI has no positions of its own, so each Step falls
// back to the viewed Space, then the overview, then the calculated placement
const flowPositions = (
  flowId: string,
  spaceId?: string | null
): NodePositions => ({
  ...loadNodePositions({ type: "overview" }),
  ...(spaceId ? loadNodePositions({ type: "space", spaceId }) : {}),
  ...loadNodePositions({ type: "flow", flowId }),
});
