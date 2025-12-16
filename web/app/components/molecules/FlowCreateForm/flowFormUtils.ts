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
    return JSON.parse(stateString);
  } catch {
    return {};
  }
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
