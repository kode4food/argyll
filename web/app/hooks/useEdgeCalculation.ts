import { useMemo } from "react";
import { Edge } from "@xyflow/react";
import { Step } from "@/app/api";
import { STEP_LAYOUT, EDGE_COLORS } from "@/constants/layout";
import {
  buildOutputProducerMap,
  listStepInputs,
} from "@/utils/stepDependencyGraph";

type StepInput = ReturnType<typeof listStepInputs>[number];

interface EdgeConnection {
  fromStepID: string;
  toStep: Step;
  input: StepInput;
}

interface EdgeViewState {
  previewStepIds: Set<string> | null | undefined;
  focusedAttributeName: string | null | undefined;
}

const buildEdge = (
  connection: EdgeConnection,
  viewState: EdgeViewState
): Edge => {
  const { fromStepID, toStep, input } = connection;
  const { previewStepIds, focusedAttributeName } = viewState;
  const isInPlan = previewStepIds
    ? previewStepIds.has(fromStepID) && previewStepIds.has(toStep.id)
    : false;
  const isOutOfPlan = !!previewStepIds && !isInPlan;
  const isFocusedAttribute = focusedAttributeName === input.name;

  const strokeColor = isOutOfPlan
    ? EDGE_COLORS.GRAYED
    : input.isOptional
      ? EDGE_COLORS.OPTIONAL
      : EDGE_COLORS.REQUIRED;

  const baseZIndex = isInPlan
    ? STEP_LAYOUT.EDGE_FOCUSED_Z_INDEX
    : STEP_LAYOUT.EDGE_Z_INDEX;

  return {
    id: `${fromStepID}-${toStep.id}-${input.name}`,
    source: fromStepID,
    target: toStep.id,
    sourceHandle: `output-${input.name}`,
    targetHandle: `input-${input.isOptional ? "optional" : "required"}-${input.name}`,
    type: "smoothstep",
    style: {
      stroke: strokeColor,
      strokeWidth: STEP_LAYOUT.EDGE_WIDTH,
      strokeDasharray: input.isOptional ? STEP_LAYOUT.DASH_PATTERN : undefined,
    },
    markerEnd: {
      type: "arrow" as const,
      color: strokeColor,
      strokeWidth: STEP_LAYOUT.EDGE_WIDTH - 0.5,
    },
    zIndex: input.isOptional ? baseZIndex : baseZIndex + 1,
    className:
      focusedAttributeName && isFocusedAttribute && !isOutOfPlan
        ? "edge-focused-animated"
        : undefined,
  };
};

export const useEdgeCalculation = (
  visibleSteps: Step[],
  previewStepIds?: Set<string> | null,
  focusedAttributeName?: string | null
) => {
  return useMemo(() => {
    const edges: Edge[] = [];
    const producerMap = buildOutputProducerMap(visibleSteps);

    visibleSteps.forEach((toStep) => {
      listStepInputs(toStep).forEach((input) => {
        const producerIDs = producerMap.get(input.name) || [];
        producerIDs.forEach((fromStepID) => {
          if (fromStepID !== toStep.id) {
            edges.push(
              buildEdge(
                { fromStepID, toStep, input },
                { previewStepIds, focusedAttributeName }
              )
            );
          }
        });
      });
    });

    return edges;
  }, [visibleSteps, previewStepIds, focusedAttributeName]);
};
