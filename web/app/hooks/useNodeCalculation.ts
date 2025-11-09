import React, { useMemo } from "react";
import { Node } from "@xyflow/react";
import {
  Step,
  WorkflowContext,
  ExecutionResult,
  ExecutionPlan,
  AttributeRole,
} from "../api";
import { STEP_LAYOUT } from "@/constants/layout";
import { loadNodePositions } from "@/utils/nodePositioning";

const calculateSectionHeight = (argCount: number): number => {
  if (argCount === 0) return 0;
  return STEP_LAYOUT.SECTION_HEIGHT + argCount * STEP_LAYOUT.ARG_LINE_HEIGHT;
};

export const useNodeCalculation = (
  visibleSteps: Step[],
  selectedStep: string | null,
  workflowData?: WorkflowContext | null,
  executions?: ExecutionResult[],
  previewPlan?: ExecutionPlan | null,
  previewStepIds?: Set<string> | null,
  onStepClick?: (stepId: string) => void,
  resolvedAttributes?: string[],
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>,
  disableEdit?: boolean
) => {
  return useMemo(() => {
    const savedPositions = loadNodePositions();

    const activeStepIds =
      previewStepIds ||
      (workflowData?.plan
        ? new Set(Object.keys(workflowData.plan.steps))
        : null);
    let startingPoints = new Set<string>();

    if (activeStepIds) {
      const withDeps = new Set<string>();

      visibleSteps.forEach((toStep) => {
        if (!activeStepIds.has(toStep.id)) return;

        const allInputNames = Object.entries(toStep.attributes || {})
          .filter(
            ([_, spec]) =>
              spec.role === AttributeRole.Required ||
              spec.role === AttributeRole.Optional
          )
          .map(([name]) => name)
          .sort();

        allInputNames.forEach((inputName) => {
          visibleSteps.forEach((fromStep) => {
            if (
              fromStep.id !== toStep.id &&
              activeStepIds.has(fromStep.id) &&
              fromStep.attributes &&
              Object.entries(fromStep.attributes).some(
                ([name, spec]) =>
                  name === inputName && spec.role === AttributeRole.Output
              )
            ) {
              withDeps.add(toStep.id);
            }
          });
        });
      });

      activeStepIds.forEach((stepId) => {
        if (!withDeps.has(stepId)) {
          startingPoints.add(stepId);
        }
      });
    }

    const dependencies = new Map<string, string[]>();
    const dependents = new Map<string, string[]>();

    visibleSteps.forEach((step) => {
      dependencies.set(step.id, []);
      dependents.set(step.id, []);
    });

    visibleSteps.forEach((toStep) => {
      const allInputNames = Object.entries(toStep.attributes || {})
        .filter(
          ([_, spec]) =>
            spec.role === AttributeRole.Required ||
            spec.role === AttributeRole.Optional
        )
        .map(([name]) => name)
        .sort();

      allInputNames.forEach((inputName) => {
        visibleSteps.forEach((fromStep) => {
          if (
            fromStep.id !== toStep.id &&
            fromStep.attributes &&
            Object.entries(fromStep.attributes).some(
              ([name, spec]) =>
                name === inputName && spec.role === AttributeRole.Output
            )
          ) {
            dependencies.get(toStep.id)?.push(fromStep.id);
            dependents.get(fromStep.id)?.push(toStep.id);
          }
        });
      });
    });

    const levels = new Map<string, number>();
    const visited = new Set<string>();

    const calculateLevel = (stepId: string): number => {
      if (levels.has(stepId)) return levels.get(stepId)!;
      if (visited.has(stepId)) return 0;

      visited.add(stepId);
      const deps = dependencies.get(stepId) || [];

      if (deps.length === 0) {
        levels.set(stepId, 0);
        return 0;
      }

      const maxDepLevel = Math.max(...deps.map((dep) => calculateLevel(dep)));
      const level = maxDepLevel + 1;
      levels.set(stepId, level);
      return level;
    };

    visibleSteps.forEach((step) => calculateLevel(step.id));

    const levelGroups = new Map<number, string[]>();
    levels.forEach((level, stepId) => {
      if (!levelGroups.has(level)) levelGroups.set(level, []);
      levelGroups.get(level)?.push(stepId);
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

        const requiredCount = Object.values(step.attributes || {}).filter(
          (spec) => spec.role === AttributeRole.Required
        ).length;
        const optionalCount = Object.values(step.attributes || {}).filter(
          (spec) => spec.role === AttributeRole.Optional
        ).length;
        const outputCount = Object.values(step.attributes || {}).filter(
          (spec) => spec.role === AttributeRole.Output
        ).length;

        const widgetHeight =
          STEP_LAYOUT.WIDGET_BASE_HEIGHT +
          calculateSectionHeight(requiredCount) +
          calculateSectionHeight(optionalCount) +
          calculateSectionHeight(outputCount);

        const col = level;
        const row = indexInLevel - (levelSize - 1) / 2;

        position = {
          x: col * STEP_LAYOUT.HORIZONTAL_SPACING,
          y:
            row * (widgetHeight + STEP_LAYOUT.VERTICAL_SPACING) +
            STEP_LAYOUT.VERTICAL_OFFSET,
        };
      }

      const isInPreviewPlan = !activeStepIds || activeStepIds.has(step.id);
      const isPreviewMode = !!previewPlan || !!previewStepIds;

      const node: Node = {
        id: step.id,
        position,
        data: {
          step,
          selected: selectedStep === step.id,
          onStepClick,
          workflowData,
          executions,
          resolvedAttributes,
          isGoalStep:
            workflowData?.plan?.goals?.includes(step.id) ||
            previewPlan?.goals?.includes(step.id),
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
    selectedStep,
    workflowData,
    executions,
    previewPlan,
    previewStepIds,
    onStepClick,
    resolvedAttributes,
    disableEdit,
    diagramContainerRef,
  ]);
};
