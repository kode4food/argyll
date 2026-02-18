import { AttributeRole } from "@/app/api";
import { STEP_LAYOUT } from "@/constants/layout";

export interface RoleCounts {
  required: number;
  optional: number;
  output: number;
}

export type AttributeSpecLike = {
  role?: AttributeRole;
};

export const calculateSectionHeight = (argCount: number): number => {
  if (argCount === 0) {
    return 0;
  }

  return STEP_LAYOUT.SECTION_HEIGHT + argCount * STEP_LAYOUT.ARG_LINE_HEIGHT;
};

export const countRoleAttributes = (
  attributes?: Record<string, AttributeSpecLike>
): RoleCounts => {
  const counts: RoleCounts = {
    required: 0,
    optional: 0,
    output: 0,
  };

  Object.values(attributes || {}).forEach((spec) => {
    if (spec.role === AttributeRole.Required) {
      counts.required += 1;
      return;
    }
    if (spec.role === AttributeRole.Optional) {
      counts.optional += 1;
      return;
    }
    if (spec.role === AttributeRole.Output) {
      counts.output += 1;
    }
  });

  return counts;
};

export const calculateWidgetHeightFromAttributes = (
  attributes?: Record<string, AttributeSpecLike>
): number => {
  const counts = countRoleAttributes(attributes);
  return (
    STEP_LAYOUT.WIDGET_BASE_HEIGHT +
    calculateSectionHeight(counts.required) +
    calculateSectionHeight(counts.optional) +
    calculateSectionHeight(counts.output)
  );
};
