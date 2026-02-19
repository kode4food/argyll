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

export function parseInputValue(rawValue: string): any {
  if (rawValue.trim() === "") {
    return undefined;
  }

  try {
    return JSON.parse(rawValue);
  } catch {
    return rawValue;
  }
}

export function buildInitialStateInputValues(
  initialState: string,
  inputNames: string[]
): Record<string, string> {
  const parsed = safeParseState(initialState);
  const values: Record<string, string> = {};

  inputNames.forEach((name) => {
    values[name] = formatInputValue(parsed[name]);
  });

  return values;
}

export function buildInitialStateFromInputValues(
  inputValues: Record<string, string>,
  inputNames: string[]
): Record<string, any> {
  const nextState: Record<string, any> = {};

  inputNames.forEach((name) => {
    const parsedValue = parseInputValue(inputValues[name] || "");
    if (parsedValue !== undefined) {
      nextState[name] = parsedValue;
    }
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
