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
