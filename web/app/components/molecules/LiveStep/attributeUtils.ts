import { ExecutionResult, AttributeSpec, AttributeRole } from "@/app/api";
import { type AttributeModifier } from "@/utils/stepUtils";

export interface ArgValueResult {
  hasValue: boolean;
  value: any;
}

export type ArgType = "required" | "optional" | "const" | "output";

export interface UnifiedArg {
  name: string;
  type: string;
  argType: "required" | "optional" | "const" | "output";
  spec: AttributeSpec;
  modifiers: AttributeModifier[];
}

export interface StatusBadgeContext {
  isSatisfied: boolean;
  isAvailable?: boolean;
  executionStatus?: string;
  isUnsatisfied?: boolean;
  isWinner?: boolean;
  isProvidedByUpstream?: boolean;
  wasDefaulted?: boolean;
}

export const getInputName = (name: string, spec: AttributeSpec): string => {
  if (spec.role === AttributeRole.Required) {
    return spec.required?.mapping?.name || name;
  }
  if (spec.role === AttributeRole.Optional) {
    return spec.optional?.mapping?.name || name;
  }
  return name;
};

export const getExecutionInputName = (arg: UnifiedArg): string =>
  getInputName(arg.name, arg.spec);

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

export const getAttributeTooltipTitle = (
  argType: ArgType,
  wasDefaulted?: boolean
): string => {
  switch (argType) {
    case "required":
      return "liveStep.inputValue";
    case "optional":
      return wasDefaulted ? "liveStep.defaultValue" : "liveStep.inputValue";
    case "const":
      return "liveStep.defaultValue";
    case "output":
      return "liveStep.outputValue";
  }
};

export const getAttributeValue = (
  arg: UnifiedArg,
  execution?: ExecutionResult
): ArgValueResult => {
  if (arg.argType === "output") {
    if (execution?.outputs && arg.name in execution.outputs) {
      return { hasValue: true, value: execution.outputs[arg.name] };
    }
    return { hasValue: false, value: undefined };
  }

  const inputName = getExecutionInputName(arg);
  if (execution?.inputs && inputName in execution.inputs) {
    return { hasValue: true, value: execution.inputs[inputName] };
  }
  return { hasValue: false, value: undefined };
};
