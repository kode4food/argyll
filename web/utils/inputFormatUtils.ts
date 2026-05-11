import { parseInputValue, parseInputValues } from "./inputParseUtils";

const DELIMITED_SPECIAL_CHARS = /[,"\\]/;

function shouldQuoteDelimitedString(value: string): boolean {
  return (
    value === "" ||
    value.trim() !== value ||
    DELIMITED_SPECIAL_CHARS.test(value)
  );
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

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value) &&
    Object.getPrototypeOf(value) === Object.prototype
  );
}

function valuesEqual(left: unknown, right: unknown): boolean {
  if (left === right) return true;
  if (
    typeof left === "number" &&
    typeof right === "number" &&
    Number.isNaN(left) &&
    Number.isNaN(right)
  )
    return true;

  if (Array.isArray(left) && Array.isArray(right)) {
    return (
      left.length === right.length &&
      left.every((value, index) => valuesEqual(value, right[index]))
    );
  }

  if (isPlainObject(left) && isPlainObject(right)) {
    const leftKeys = Object.keys(left).sort();
    const rightKeys = Object.keys(right).sort();
    return (
      leftKeys.length === rightKeys.length &&
      leftKeys.every(
        (key, i) =>
          key === rightKeys[i] && valuesEqual(left[key], right[rightKeys[i]])
      )
    );
  }

  return false;
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
