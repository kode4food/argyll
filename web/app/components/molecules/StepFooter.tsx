import React from "react";
import { Code2, FileCode2, Globe, Webhook } from "lucide-react";
import {
  Step,
  ExecutionResult,
  HealthStatus,
  SCRIPT_LANGUAGE_ALE,
} from "../../api";
import { getHealthIconClass, getHealthStatusText } from "@/utils/healthUtils";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "../../hooks/useStepProgress";
import Tooltip from "../atoms/Tooltip";
import TooltipSection from "../atoms/TooltipSection";
import tooltipStyles from "../atoms/TooltipSection.module.css";
import HealthDot from "../atoms/HealthDot";

interface StepFooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
  flowId?: string;
  execution?: ExecutionResult;
}

const StepFooter: React.FC<StepFooterProps> = ({
  step,
  healthStatus,
  healthError,
  flowId,
  execution,
}) => {
  const progressState = useStepProgress(step.id, flowId, execution);

  const useProgress = flowId && progressState.flowId === flowId;

  const healthIconClass = getHealthIconClass(healthStatus, step.type);
  const healthText = getHealthStatusText(healthStatus, healthError);

  const ProgressIcon = getProgressIcon(progressState.status);

  const getStepInfoDisplay = () => {
    if (step.type === "script" && step.script) {
      const ScriptIcon =
        step.script.language === SCRIPT_LANGUAGE_ALE ? FileCode2 : Code2;
      const scriptPreview = step.script.script.replace(/\n/g, " ");
      return (
        <div className="flex min-w-0 flex-1 items-center gap-1">
          <ScriptIcon className="step-type-icon h-3.5 w-3.5 flex-shrink-0" />
          <span className="step-endpoint script">{scriptPreview}</span>
        </div>
      );
    }
    if (step.http) {
      const HttpIcon = step.type === "async" ? Webhook : Globe;
      return (
        <div className="flex min-w-0 flex-1 items-center gap-1">
          <HttpIcon className="step-type-icon h-3.5 w-3.5 flex-shrink-0" />
          <span className="step-endpoint">{step.http.endpoint}</span>
        </div>
      );
    }
    return null;
  };

  const renderTooltipContent = () => {
    const sections = [];

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
            <div className="step-progress-details">
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
        const skipReason = step.predicate
          ? "Step skipped because predicate evaluated to false"
          : "Step skipped because required inputs are unavailable due to failed or skipped upstream steps";

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
        const scriptPreview = step.script.script
          .split("\n")
          .slice(0, 5)
          .join("\n");
        const lineCount = step.script.script.split("\n").length;

        sections.push(
          <TooltipSection
            key="script"
            title={`Script Preview (${step.script.language})`}
          >
            <div className={tooltipStyles.valueCode}>
              {scriptPreview}
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
          icon={<HealthDot className={healthIconClass} />}
        >
          {healthText}
        </TooltipSection>
      );
    }

    return <>{sections}</>;
  };

  return (
    <Tooltip
      trigger={
        <div className="step-footer flex-between">
          {getStepInfoDisplay()}
          <div className="step-footer-actions">
            <div className="step-health-status">
              {useProgress ? (
                <div className="step-progress-container">
                  <ProgressIcon
                    className={`progress-icon ${progressState.status || "pending"}`}
                  />
                  {progressState.status === "active" &&
                    progressState.workItems &&
                    progressState.workItems.total > 1 && (
                      <span className="step-progress-counter">
                        (
                        {progressState.workItems.completed +
                          progressState.workItems.failed}
                        /{progressState.workItems.total})
                      </span>
                    )}
                </div>
              ) : (
                <HealthDot className={healthIconClass} />
              )}
            </div>
          </div>
        </div>
      }
    >
      {renderTooltipContent()}
    </Tooltip>
  );
};

export default StepFooter;
