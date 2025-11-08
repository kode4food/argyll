import { useMemo } from "react";
import { Edge } from "@xyflow/react";
import { Step, AttributeRole } from "../api";
import { STEP_LAYOUT, EDGE_COLORS } from "@/constants/layout";

export const useEdgeCalculation = (
  visibleSteps: Step[],
  previewStepIds?: Set<string> | null
) => {
  return useMemo(() => {
    const edges: Edge[] = [];

    visibleSteps.forEach((toStep) => {
      const allInputs = Object.entries(toStep.attributes || {})
        .filter(
          ([_, attr]) =>
            attr.role === AttributeRole.Required ||
            attr.role === AttributeRole.Optional
        )
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([name, attr]) => ({
          name,
          isOptional: attr.role === AttributeRole.Optional,
        }));

      allInputs.forEach((input) => {
        visibleSteps.forEach((fromStep) => {
          const hasOutput = Object.entries(fromStep.attributes || {}).some(
            ([name, attr]) =>
              attr.role === AttributeRole.Output && name === input.name
          );
          if (fromStep.id !== toStep.id && hasOutput) {
            const outputArg = input.name;
            const isInPlan = previewStepIds
              ? previewStepIds.has(fromStep.id) && previewStepIds.has(toStep.id)
              : false;

            const strokeColor =
              previewStepIds && !isInPlan
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

            edges.push({
              id: `${fromStep.id}-${toStep.id}-${outputArg}`,
              source: fromStep.id,
              target: toStep.id,
              sourceHandle: `output-${outputArg}`,
              targetHandle: `input-${input.isOptional ? "optional" : "required"}-${input.name}`,
              type: "smoothstep",
              style: edgeStyle,
              markerEnd: {
                type: "arrow",
                color: strokeColor,
              },
              zIndex: isInPlan ? 1000 : 1,
            });
          }
        });
      });
    });

    return edges;
  }, [visibleSteps, previewStepIds]);
};
