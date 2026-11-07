import { ExecutionResult, AttributeSpec, AttributeRole } from "@/app/api";
import { type ArgType } from "@/utils/iconRegistry";
import { type AttributeModifier } from "@/utils/stepUtils";

export type { ArgType };

export interface ArgValueResult {
  hasValue: boolean;
  value: any;
}

export interface UnifiedArg {
  name: string;
  argType: ArgType;
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
      return "liveStep.constValue";
    case "meta":
      return "liveStep.inputValue";
    case "feed":
      return "liveStep.feedValue";
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
