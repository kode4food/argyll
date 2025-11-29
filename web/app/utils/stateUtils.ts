import { AttributeType, ExecutionPlan, Step } from "../api";

export const getDefaultValueForType = (type?: AttributeType): any => {
  if (!type) return "";

  switch (type) {
    case AttributeType.Boolean:
      return false;
    case AttributeType.Number:
      return 0;
    case AttributeType.String:
      return "";
    case AttributeType.Object:
      return {};
    case AttributeType.Array:
      return [];
    case AttributeType.Null:
      return null;
    case AttributeType.Any:
    default:
      return "";
  }
};

export const isDefaultValue = (value: any, type?: AttributeType): boolean => {
  if (type === undefined) {
    return isUntypedDefaultValue(value);
  }

  const defaultForType = getDefaultValueForType(type);

  // For objects and arrays, need deep comparison
  if (typeof defaultForType === "object") {
    if (Array.isArray(defaultForType)) {
      return Array.isArray(value) && value.length === 0;
    }
    return typeof value === "object" &&
           value !== null &&
           !Array.isArray(value) &&
           Object.keys(value).length === 0;
  }

  return value === defaultForType;
};

export const isUntypedDefaultValue = (value: any): boolean => {
  if (value === "" || value === null) return true;
  if (value === false) return true;
  if (value === 0) return true;
  if (typeof value === "object") {
    if (Array.isArray(value)) {
      return value.length === 0;
    }
    return Object.keys(value).length === 0;
  }
  return false;
};

export const parseState = (stateJson: string): Record<string, any> => {
  try {
    return JSON.parse(stateJson);
  } catch {
    return {};
  }
};

export const filterDefaultValues = (
  state: Record<string, any>,
  steps?: Step[]
): Record<string, any> => {
  const filtered: Record<string, any> = {};

  Object.keys(state).forEach((key) => {
    let attributeType: AttributeType | undefined;

    // Find the type from the global step registry
    if (steps) {
      for (const step of steps) {
        if (step.attributes?.[key]) {
          attributeType = step.attributes[key].type;
          break;
        }
      }
    }

    const isDefault = isDefaultValue(state[key], attributeType);

    if (!isDefault) {
      filtered[key] = state[key];
    }
  });

  return filtered;
};

export const addRequiredDefaults = (
  state: Record<string, any>,
  executionPlan: ExecutionPlan
): Record<string, any> => {
  const result = { ...state };

  (executionPlan.required || []).forEach((name) => {
    if (!(name in result)) {
      // Find the attribute type from any step that declares it
      let attributeType: AttributeType | undefined;
      for (const step of Object.values(executionPlan.steps || {})) {
        if (step.attributes?.[name]) {
          attributeType = step.attributes[name].type;
          break;
        }
      }
      result[name] = getDefaultValueForType(attributeType);
    }
  });

  return result;
};
