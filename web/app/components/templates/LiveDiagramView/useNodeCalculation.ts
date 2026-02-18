import React, { useMemo } from "react";
import { Node } from "@xyflow/react";
import { Step, FlowContext, ExecutionResult } from "@/app/api";
import { STEP_LAYOUT } from "@/constants/layout";
import { loadNodePositions } from "@/utils/nodePositioning";
import {
  buildOutputProducerMap,
  buildStepGraph,
  calculateStepLevels,
  countStepRoleAttributes,
} from "@/utils/stepDependencyGraph";

const calculateSectionHeight = (argCount: number): number => {
  if (argCount === 0) return 0;
  return STEP_LAYOUT.SECTION_HEIGHT + argCount * STEP_LAYOUT.ARG_LINE_HEIGHT;
};

export const useNodeCalculation = (
  visibleSteps: Step[],
  flowData?: FlowContext | null,
  executions?: ExecutionResult[],
  resolvedAttributes?: string[],
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>,
  disableEdit?: boolean
) => {
  return useMemo(() => {
    const savedPositions = flowData?.id
      ? loadNodePositions({ type: "flow", flowId: flowData.id })
      : loadNodePositions();
    const activeStepIDs = new Set(visibleSteps.map((step) => step.id));
    const producerMap = buildOutputProducerMap(visibleSteps);
    const { dependencies, stepsWithDependencies } = buildStepGraph(
      visibleSteps,
      producerMap,
      activeStepIDs
    );
    const levels = calculateStepLevels(visibleSteps, dependencies);
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
        const roleCounts = countStepRoleAttributes(step);

        const widgetHeight =
          STEP_LAYOUT.WIDGET_BASE_HEIGHT +
          calculateSectionHeight(roleCounts.required) +
          calculateSectionHeight(roleCounts.optional) +
          calculateSectionHeight(roleCounts.output);

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
          isGoalStep: flowData?.plan?.goals?.includes(step.id),
          isStartingPoint: startingPoints.has(step.id),
          diagramContainerRef,
          disableEdit,
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
    disableEdit,
    diagramContainerRef,
  ]);
};
