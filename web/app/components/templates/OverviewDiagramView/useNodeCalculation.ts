import React, { useMemo } from "react";
import { Node } from "@xyflow/react";
import { Step, ExecutionPlan } from "@/app/api";
import { STEP_LAYOUT } from "@/constants/layout";
import { loadNodePositions } from "@/utils/nodePositioning";
import {
  buildOutputProducerMap,
  buildStepGraph,
  calculateStepLevels,
} from "@/utils/stepDependencyGraph";
import { calculateWidgetHeightFromAttributes } from "@/utils/stepLayout";

export const useNodeCalculation = (
  visibleSteps: Step[],
  selectedStepIds: string[],
  previewPlan?: ExecutionPlan | null,
  previewStepIds?: Set<string> | null,
  onStepClick?: (stepId: string, options?: { additive?: boolean }) => void,
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>,
  disableEdit?: boolean
) => {
  return useMemo(() => {
    const savedPositions = loadNodePositions();
    const activeStepIDs = previewStepIds || null;
    const producerMap = buildOutputProducerMap(visibleSteps);
    const { dependencies, stepsWithDependencies } = buildStepGraph(
      visibleSteps,
      producerMap,
      activeStepIDs
    );
    const levels = calculateStepLevels(visibleSteps, dependencies);
    const startingPoints = new Set<string>();

    if (activeStepIDs) {
      activeStepIDs.forEach((stepID) => {
        if (!stepsWithDependencies.has(stepID)) {
          startingPoints.add(stepID);
        }
      });
    }

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

      const isInPreviewPlan = !activeStepIDs || activeStepIDs.has(step.id);
      const isPreviewMode = !!previewPlan || !!previewStepIds;

      const node: Node = {
        id: step.id,
        position,
        data: {
          step,
          selected: selectedStepIds.includes(step.id),
          onStepClick,
          isGoalStep: previewPlan?.goals?.includes(step.id),
          isInPreviewPlan,
          isPreviewMode,
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
    selectedStepIds,
    previewPlan,
    previewStepIds,
    onStepClick,
    disableEdit,
    diagramContainerRef,
  ]);
};
