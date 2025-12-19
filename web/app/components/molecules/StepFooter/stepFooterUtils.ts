import { Step, SCRIPT_LANGUAGE_ALE } from "../../../api";
import { Code2, FileCode2, Globe, Webhook } from "lucide-react";
import React from "react";

export interface StepDisplayInfo {
  icon: React.ComponentType<{ className?: string }>;
  text: string;
  className?: string;
}

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
 * Generates the skip reason message based on step configuration
 */
export const getSkipReason = (step: Step): string => {
  return step.predicate
    ? "Step skipped because predicate evaluated to false"
    : "Step skipped because required inputs are unavailable due to failed or skipped upstream steps";
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
