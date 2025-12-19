import { ExecutionResult, AttributeSpec } from "../../../api";

export interface ArgValueResult {
  hasValue: boolean;
  value: any;
}

export type ArgType = "required" | "optional" | "output";

export interface UnifiedArg {
  name: string;
  type: string;
  argType: "required" | "optional" | "output";
  spec: AttributeSpec;
}

export interface StatusBadgeContext {
  isSatisfied: boolean;
  executionStatus?: string;
  isWinner?: boolean;
  isProvidedByUpstream?: boolean;
  wasDefaulted?: boolean;
}

/**
 * Formats a value for display in tooltips
 * Handles null, undefined, strings, objects, and other types
 */
export const formatAttributeValue = (val: any): string => {
  if (val === null) return "null";
  if (val === undefined) return "undefined";
  if (typeof val === "string") return `"${val}"`;
  if (typeof val === "object") {
    try {
      return JSON.stringify(val, null, 2);
    } catch {
      return String(val);
    }
  }
  return String(val);
};

/**
 * Gets the tooltip title based on attribute type and whether it was defaulted
 */
export const getAttributeTooltipTitle = (
  argType: ArgType,
  wasDefaulted?: boolean
): string => {
  switch (argType) {
    case "required":
      return "Input Value";
    case "optional":
      return wasDefaulted ? "Default Value" : "Input Value";
    case "output":
      return "Output Value";
  }
};

/**
 * Extracts the value for an attribute from execution results
 * Handles both inputs and outputs
 */
export const getAttributeValue = (
  arg: UnifiedArg,
  execution?: ExecutionResult
): ArgValueResult => {
  if (arg.argType === "output") {
    const hasValue = !!execution?.outputs && arg.name in execution.outputs;
    return {
      hasValue,
      value: hasValue ? execution?.outputs?.[arg.name] : undefined,
    };
  }

  const hasValue = !!execution?.inputs && arg.name in execution.inputs;
  return {
    hasValue,
    value: hasValue ? execution?.inputs?.[arg.name] : undefined,
  };
};
