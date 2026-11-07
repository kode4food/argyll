import { ExecutionPlan } from "@/app/api";

export const isDefaultValue = (value: any): boolean => {
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
  state: Record<string, any>
): Record<string, any> => {
  const filtered: Record<string, any> = {};

  Object.keys(state).forEach((key) => {
    if (state[key] === "") return;
    if (Array.isArray(state[key]) || !isDefaultValue(state[key])) {
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

  const requiredNames = [...(executionPlan.required || [])].sort((a, b) =>
    a.localeCompare(b)
  );

  requiredNames.forEach((name) => {
    if (!(name in result)) {
      result[name] = "";
    }
  });

  return result;
};
