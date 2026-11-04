// A value JSON cannot hold, such as undefined or a cycle, shows as its string
function stringify(value: unknown, indent?: number): string {
  try {
    return JSON.stringify(value, null, indent) ?? String(value);
  } catch {
    return String(value);
  }
}

export const formatJSON = (value: unknown): string => stringify(value, 2);

export const formatCompactJSON = (value: unknown): string => stringify(value);
