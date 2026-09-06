import { SCRIPT_LANGUAGE_LUA } from "@/app/api";

export interface ScriptPreviewData {
  preview: string;
  lineCount: number;
}

const LEADING_LUA_RETURN = /^[ \t]*return[ \t]+/;

/**
 * Drops the leading `return` a Lua script needs but a one-line footer does
 * not. Tooltips show the script as written
 */
export const stripLuaReturn = (script: string, language?: string): string => {
  if (language !== SCRIPT_LANGUAGE_LUA) {
    return script;
  }
  return script.replace(LEADING_LUA_RETURN, "");
};

/**
 * Formats a script onto one line for inline display, collapsing every run of
 * whitespace to a single space. Spacing inside string literals collapses too,
 * which the tooltip's as-written copy makes acceptable
 */
export const formatScriptPreview = (
  script: string,
  language?: string
): string => {
  return stripLuaReturn(script, language).replace(/\s+/g, " ").trim();
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
