import { AttributeSpec, AttributeRole } from "../../../api";

export interface HandlePosition {
  id: string;
  top: number;
  argName: string;
  handleType: "input" | "output";
}

export interface HandlePositions {
  required: HandlePosition[];
  optional: HandlePosition[];
  output: HandlePosition[];
}

export interface AttributeGroups {
  required: string[];
  optional: string[];
  output: string[];
}

/**
 * Groups attributes by their role (required, optional, output)
 * Returns sorted arrays for each type
 */
export const groupAttributesByRole = (
  attributes: Record<string, AttributeSpec>
): AttributeGroups => {
  const sortedAttrs = Object.entries(attributes || {}).sort(([a], [b]) =>
    a.localeCompare(b)
  );

  return {
    required: sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Required)
      .map(([name]) => name),
    optional: sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Optional)
      .map(([name]) => name),
    output: sortedAttrs
      .filter(([_, spec]) => spec.role === AttributeRole.Output)
      .map(([name]) => name),
  };
};

/**
 * Generates a unique handle ID based on type and name
 */
export const generateHandleId = (type: string, name: string): string => {
  return type === "output" ? `output-${name}` : `input-${type}-${name}`;
};

/**
 * Builds provenance map from flow state
 * Maps attribute name to the step ID that produced it
 */
export const buildProvenanceMap = (
  flowState?: Record<string, any>
): Map<string, string> => {
  const map = new Map<string, string>();
  if (flowState) {
    Object.entries(flowState).forEach(([attrName, attrValue]) => {
      if (attrValue?.step) {
        map.set(attrName, attrValue.step);
      }
    });
  }
  return map;
};

/**
 * Calculates which arguments are satisfied based on resolved attributes
 * Only considers required and optional arguments
 */
export const calculateSatisfiedArgs = (
  attributes: Record<string, AttributeSpec>,
  resolved: Set<string>
): Set<string> => {
  const set = new Set<string>();
  Object.entries(attributes || {}).forEach(([argName, spec]) => {
    if (
      (spec.role === AttributeRole.Required ||
        spec.role === AttributeRole.Optional) &&
      resolved.has(argName)
    ) {
      set.add(argName);
    }
  });
  return set;
};
