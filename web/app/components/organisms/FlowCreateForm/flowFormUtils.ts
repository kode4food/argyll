import { formatInputValues } from "@/utils/inputFormatUtils";
import { parseInputValues } from "@/utils/inputParseUtils";

export { parseInputValue, parseInputValues } from "@/utils/inputParseUtils";
export {
  formatInputValue,
  formatInputValues,
  isAtDefaultValue,
} from "@/utils/inputFormatUtils";

export function hasScrollOverflow(element: HTMLElement): {
  hasOverflow: boolean;
  atTop: boolean;
  atBottom: boolean;
} {
  const { scrollTop, scrollHeight, clientHeight } = element;
  const hasOverflow = scrollHeight > clientHeight;

  if (!hasOverflow) {
    return { hasOverflow: false, atTop: true, atBottom: true };
  }

  const atTop = scrollTop <= 1;
  const atBottom = scrollTop >= scrollHeight - clientHeight - 1;

  return { hasOverflow, atTop, atBottom };
}

export function safeParseState(stateString: string): Record<string, any> {
  try {
    const parsed = JSON.parse(stateString);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return parsed;
  } catch {
    return {};
  }
}

export type FlowInputStatus =
  | "requiredMissing"
  | "requiredSatisfied"
  | "optionalMissing"
  | "optionalSatisfied"
  | "outputSatisfied"
  | "unreachable";

export function getFlowInputStatus(
  attr: {
    required: boolean;
    defaultValue?: string;
    unreachable?: boolean;
    satisfiedByOutput?: boolean;
  },
  rawValue: string
): FlowInputStatus {
  if (attr.unreachable) {
    return "unreachable";
  }
  if (attr.satisfiedByOutput) {
    return "outputSatisfied";
  }
  if (parseInputValues(rawValue).length === 0) {
    return attr.required ? "requiredMissing" : "optionalMissing";
  }
  return attr.required ? "requiredSatisfied" : "optionalSatisfied";
}

export function buildInitialStateInputValues(
  initialState: string,
  inputNames: string[]
): Record<string, string> {
  const parsed = safeParseState(initialState);
  const values: Record<string, string> = {};

  inputNames.forEach((name) => {
    values[name] = formatInputValues(parsed[name]);
  });

  return values;
}

export function buildInitialStateFromInputValues(
  inputValues: Record<string, string>,
  inputNames: string[]
): Record<string, any> {
  const nextState: Record<string, any> = {};

  inputNames.forEach((name) => {
    nextState[name] = parseInputValues(inputValues[name] || "");
  });

  return nextState;
}

export function validateJsonString(jsonString: string): string | null {
  try {
    JSON.parse(jsonString);
    return null;
  } catch (error: any) {
    return error.message;
  }
}

export interface ItemClassNames {
  base: string;
  selected: string;
  disabled: string;
}

export function buildItemClassName(
  isSelected: boolean,
  isDisabled: boolean,
  classNames: ItemClassNames
): string {
  return [
    classNames.base,
    isSelected && classNames.selected,
    isDisabled && classNames.disabled,
  ]
    .filter(Boolean)
    .join(" ");
}
