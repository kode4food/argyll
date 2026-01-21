export interface ScriptPreviewData {
  preview: string;
  lineCount: number;
}

/**
 * Formats a script by replacing newlines with spaces for inline display
 */
export const formatScriptPreview = (script: string): string => {
  return script.replace(/\n/g, " ");
};

/**
 * Formats a script for tooltip display, showing first N lines and total count
 */
export const formatScriptForTooltip = (
  script: string,
  maxLines: number = 5
): ScriptPreviewData => {
  const lines = script.split("\n");
  const preview = lines.slice(0, maxLines).join("\n");
  return {
    preview,
    lineCount: lines.length,
  };
};
