/**
 * Checks if a timestamp string represents a valid date (not Go's zero value)
 */
export function isValidTimestamp(
  timestamp: string | undefined | null
): boolean {
  if (!timestamp) return false;
  return new Date(timestamp).getFullYear() > 1;
}
