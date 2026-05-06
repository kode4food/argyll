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

export function formatInputValue(value: any): string {
  if (value === undefined || value === null) {
    return "";
  }

  if (typeof value === "string") {
    return value;
  }

  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function shouldQuoteDelimitedString(value: string): boolean {
  return (
    value === "" ||
    value.trim() !== value ||
    value.includes(",") ||
    value.includes('"') ||
    value.includes("\\")
  );
}

export function formatInputValues(values: any): string {
  if (Array.isArray(values)) {
    return values
      .map((value) => {
        if (typeof value === "string" && shouldQuoteDelimitedString(value)) {
          return JSON.stringify(value);
        }
        return formatInputValue(value);
      })
      .join(", ");
  }

  return formatInputValue(values);
}

export function parseInputValue(rawValue: string): any {
  const trimmed = rawValue.trim();

  if (trimmed === "") {
    return undefined;
  }

  try {
    return JSON.parse(trimmed);
  } catch {
    const singleQuotedString = parseSingleQuotedString(trimmed);
    return singleQuotedString ?? rawValue;
  }
}

function parseSingleQuotedString(rawValue: string): string | undefined {
  if (
    rawValue.length < 2 ||
    !rawValue.startsWith("'") ||
    !rawValue.endsWith("'")
  ) {
    return undefined;
  }

  const inner = rawValue.slice(1, -1);
  let jsonString = '"';

  for (let i = 0; i < inner.length; i += 1) {
    const char = inner[i];
    const next = inner[i + 1];

    if (char === "\\" && next === "'") {
      jsonString += "'";
      i += 1;
      continue;
    }

    if (char === "\\" && next !== undefined) {
      jsonString += char + next;
      i += 1;
      continue;
    }

    if (char === '"') {
      jsonString += '\\"';
      continue;
    }

    jsonString += char;
  }

  jsonString += '"';

  try {
    return JSON.parse(jsonString);
  } catch {
    return undefined;
  }
}

function splitDelimitedInputValues(rawValue: string): string[] {
  const parts: string[] = [];
  let current = "";
  let quote: '"' | "'" | null = null;
  let escaped = false;
  let depth = 0;

  for (const char of rawValue) {
    if (quote) {
      if (escaped) {
        escaped = false;
      } else if (char === "\\") {
        escaped = true;
      } else if (char === quote) {
        quote = null;
      }
    } else if (char === '"' || char === "'") {
      quote = char;
    } else if (char === "{" || char === "[") {
      depth += 1;
    } else if (char === "}" || char === "]") {
      depth = Math.max(0, depth - 1);
    } else if (char === "," && depth === 0) {
      parts.push(current.trim());
      current = "";
      continue;
    }

    current += char;
  }

  parts.push(current.trim());
  return parts;
}

export function parseInputValues(rawValue: string): any[] {
  const trimmed = rawValue.trim();

  if (trimmed === "") {
    return [];
  }

  try {
    return JSON.parse(`[${trimmed}]`);
  } catch {
    return splitDelimitedInputValues(rawValue)
      .filter((part) => part !== "")
      .map((part) => parseInputValue(part));
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value) &&
    Object.getPrototypeOf(value) === Object.prototype
  );
}

function valuesEqual(left: unknown, right: unknown): boolean {
  if (left === right) {
    return true;
  }

  if (
    typeof left === "number" &&
    typeof right === "number" &&
    Number.isNaN(left) &&
    Number.isNaN(right)
  ) {
    return true;
  }

  if (Array.isArray(left) && Array.isArray(right)) {
    if (left.length !== right.length) {
      return false;
    }
    return left.every((value, index) => valuesEqual(value, right[index]));
  }

  if (isPlainObject(left) && isPlainObject(right)) {
    const leftKeys = Object.keys(left).sort();
    const rightKeys = Object.keys(right).sort();
    if (leftKeys.length !== rightKeys.length) {
      return false;
    }
    for (let i = 0; i < leftKeys.length; i += 1) {
      if (leftKeys[i] !== rightKeys[i]) {
        return false;
      }
      if (!valuesEqual(left[leftKeys[i]], right[rightKeys[i]])) {
        return false;
      }
    }
    return true;
  }

  return false;
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

export function isAtDefaultValue(
  attr: { defaultValue?: string },
  rawValue: string
): boolean {
  if (attr.defaultValue === undefined) {
    return false;
  }
  return valuesEqual(parseInputValues(rawValue), [
    parseInputValue(attr.defaultValue),
  ]);
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

export function buildItemClassName(
  isSelected: boolean,
  isDisabled: boolean,
  baseClass: string,
  selectedClass: string,
  disabledClass: string
): string {
  return [baseClass, isSelected && selectedClass, isDisabled && disabledClass]
    .filter(Boolean)
    .join(" ");
}
