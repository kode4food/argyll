import React from "react";
import { SCRIPT_LANGUAGE_ALE } from "@/app/api";
import { Code2, FileCode2, Globe, Webhook, Workflow } from "lucide-react";

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
 * Gets the appropriate script icon based on language
 */
export const getScriptIcon = (language: string): React.ComponentType => {
  return language === SCRIPT_LANGUAGE_ALE ? FileCode2 : Code2;
};

/**
 * Gets the appropriate HTTP icon based on step type
 */
export const getHttpIcon = (stepType: string): React.ComponentType => {
  return stepType === "async" ? Webhook : Globe;
};

/**
 * Gets the appropriate flow icon
 */
export const getFlowIcon = (): React.ComponentType => {
  return Workflow;
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
