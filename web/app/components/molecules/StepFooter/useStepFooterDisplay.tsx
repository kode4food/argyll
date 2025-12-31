import { useMemo } from "react";
import React from "react";
import { Step, ExecutionResult, HealthStatus } from "@/app/api";
import { getProgressIcon } from "@/utils/progressUtils";
import {
  formatScriptPreview,
  getScriptIcon,
  getHttpIcon,
  getSkipReason,
  formatScriptForTooltip,
  StepDisplayInfo,
} from "./stepFooterUtils";
import HealthDot from "../../atoms/HealthDot";
import TooltipSection from "../../atoms/TooltipSection";
import tooltipStyles from "../../atoms/TooltipSection.module.css";

export interface ProgressState {
  status: string;
  flowId?: string;
  workItems?: {
    total: number;
    completed: number;
    failed: number;
    active: number;
  };
}

export interface StepFooterDisplayData {
  displayInfo: StepDisplayInfo | null;
  tooltipSections: React.ReactElement[];
}

/**
 * Hook that prepares display data for StepFooter component
 * Handles both step info display and tooltip content generation
 */
export const useStepFooterDisplay = (
  step: Step,
  execution: ExecutionResult | undefined,
  flowId: string | undefined,
  healthStatus: HealthStatus,
  healthError: string | undefined,
  healthText: string,
  progressState: ProgressState
): StepFooterDisplayData => {
  return useMemo(() => {
    let displayInfo: StepDisplayInfo | null = null;
    if (step.type === "script" && step.script) {
      const ScriptIcon = getScriptIcon(step.script.language);
      const scriptPreview = formatScriptPreview(step.script.script);
      displayInfo = {
        icon: ScriptIcon,
        text: scriptPreview,
        className: "endpoint-script",
      };
    } else if (step.http) {
      const HttpIcon = getHttpIcon(step.type);
      displayInfo = {
        icon: HttpIcon,
        text: step.http.endpoint,
      };
    }

    const sections: React.ReactElement[] = [];

    if (execution && flowId) {
      const StatusIcon = getProgressIcon(execution.status);

      sections.push(
        <TooltipSection
          key="execution-status"
          title="Execution Status"
          icon={
            <StatusIcon
              className={`progress-icon ${execution.status || "pending"}`}
            />
          }
          bold
        >
          {execution.status.toUpperCase()}
          {progressState.workItems && progressState.workItems.total > 1 && (
            <div style={{ marginTop: "8px", fontSize: "0.875rem" }}>
              Work Items: {progressState.workItems.completed} completed,{" "}
              {progressState.workItems.failed} failed,{" "}
              {progressState.workItems.active} active (
              {progressState.workItems.completed +
                progressState.workItems.failed +
                progressState.workItems.active}
              /{progressState.workItems.total})
            </div>
          )}
        </TooltipSection>
      );

      if (execution.status === "failed" && execution.error_message) {
        sections.push(
          <TooltipSection key="error" title="Error" monospace>
            {execution.error_message}
          </TooltipSection>
        );
      }

      if (execution.status === "skipped") {
        const skipReason = getSkipReason(step);
        sections.push(
          <TooltipSection key="reason" title="Reason">
            {skipReason}
          </TooltipSection>
        );
      }

      if (execution.status === "completed" && execution.duration_ms) {
        sections.push(
          <TooltipSection key="duration" title="Duration">
            {execution.duration_ms}ms
          </TooltipSection>
        );
      }
    }

    if (!flowId) {
      if (step.type === "script" && step.script) {
        const { preview, lineCount } = formatScriptForTooltip(
          step.script.script,
          5
        );

        sections.push(
          <TooltipSection
            key="script"
            title={`Script Preview (${step.script.language})`}
          >
            <div className={tooltipStyles.valueCode}>
              {preview}
              {lineCount > 5 && (
                <div className={tooltipStyles.codeMore}>
                  ... ({lineCount - 5} more lines)
                </div>
              )}
            </div>
          </TooltipSection>
        );
      } else if (step.http) {
        sections.push(
          <TooltipSection key="endpoint" title="Endpoint URL">
            {step.http.endpoint}
          </TooltipSection>
        );

        if (step.http.health_check) {
          sections.push(
            <TooltipSection key="health-check" title="Health Check URL">
              {step.http.health_check}
            </TooltipSection>
          );
        }
      }

      sections.push(
        <TooltipSection
          key="health"
          title="Health Status"
          icon={<HealthDot status={healthStatus} />}
        >
          {healthText}
        </TooltipSection>
      );
    }

    return {
      displayInfo,
      tooltipSections: sections,
    };
  }, [step, execution, flowId, healthStatus, healthText, progressState]);
};
