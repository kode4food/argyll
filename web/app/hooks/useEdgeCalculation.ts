import { useMemo } from "react";
import { Edge } from "@xyflow/react";
import { Step } from "@/app/api";
import { STEP_LAYOUT, EDGE_COLORS } from "@/constants/layout";
import {
  buildOutputProducerMap,
  listStepInputs,
} from "@/utils/stepDependencyGraph";

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
          if (fromStepID === toStep.id) {
            return;
          }

          const isInPlan = previewStepIds
            ? previewStepIds.has(fromStepID) && previewStepIds.has(toStep.id)
            : false;
          const isFocusedAttribute = focusedAttributeName === input.name;
          const isOutOfPlan = !!previewStepIds && !isInPlan;

          const strokeColor = isOutOfPlan
            ? EDGE_COLORS.GRAYED
            : input.isOptional
              ? EDGE_COLORS.OPTIONAL
              : EDGE_COLORS.REQUIRED;

          const edgeStyle = {
            stroke: strokeColor,
            strokeWidth: STEP_LAYOUT.EDGE_WIDTH,
            strokeDasharray: input.isOptional
              ? STEP_LAYOUT.DASH_PATTERN
              : undefined,
          };
          const baseZIndex = isInPlan ? 1000 : 1;
          const edgeZIndex = input.isOptional ? baseZIndex : baseZIndex + 1;

          edges.push({
            id: `${fromStepID}-${toStep.id}-${input.name}`,
            source: fromStepID,
            target: toStep.id,
            sourceHandle: `output-${input.name}`,
            targetHandle: `input-${input.isOptional ? "optional" : "required"}-${input.name}`,
            type: "smoothstep",
            style: edgeStyle,
            markerEnd: {
              type: "arrow",
              color: strokeColor,
            },
            zIndex: edgeZIndex,
            className:
              focusedAttributeName && isFocusedAttribute
                ? "edge-focused-animated"
                : undefined,
          });
        });
      });
    });

    return edges;
  }, [visibleSteps, previewStepIds, focusedAttributeName]);
};
