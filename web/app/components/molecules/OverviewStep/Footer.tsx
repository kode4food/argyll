import React, { useMemo } from "react";
import {
  DEFAULT_HTTP_METHOD,
  Step,
  HealthStatus,
  NodeStepHealth,
} from "@/app/api";
import { getHealthIconClass } from "@/utils/healthUtils";
import Tooltip from "@/app/components/atoms/Tooltip";
import HealthDot from "@/app/components/atoms/HealthDot";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "../StepShared/StepFooter.module.css";
import { formatScriptForTooltip } from "@/utils/stepFooterUtils";
import {
  IconCompensate,
  IconEndpoint,
  IconHealthCheck,
} from "@/utils/iconRegistry";
import StepInfoDisplay from "../StepShared/StepInfoDisplay";
import { flowGoalsText, stepDisplayInfo } from "../StepShared/stepDisplayInfo";
import tooltipStyles from "@/app/components/atoms/TooltipSection/TooltipSection.module.css";
import { useT } from "@/app/i18n";

interface FooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
  healthNodes?: NodeStepHealth[];
}

const nodeHealthText = (
  t: (key: string) => string,
  healthStatus: HealthStatus,
  healthError?: string
): string => {
  return healthStatus === "healthy"
    ? t("healthStatus.healthy")
    : healthStatus === "unhealthy"
      ? healthError || t("healthStatus.unhealthy")
      : healthStatus === "unconfigured"
        ? t("healthStatus.unconfigured")
        : t("healthStatus.unknown");
};

const Footer: React.FC<FooterProps> = ({
  step,
  healthStatus,
  healthError,
  healthNodes = [],
}) => {
  const t = useT();
  const healthIconClass = getHealthIconClass(healthStatus, step.type);
  const healthText = nodeHealthText(t, healthStatus, healthError);
  const showNodeHealth = healthNodes.some(
    (nodeHealth) => nodeHealth.status !== "healthy"
  );

  const { displayInfo, tooltipSections } = useMemo(() => {
    const displayInfo = stepDisplayInfo(step);
    const goals = flowGoalsText(step);
    const sections: React.ReactElement[] = [];

    if (step.type === "script" && step.script) {
      const { preview, lineCount } = formatScriptForTooltip(
        step.script.source,
        5
      );

      sections.push(
        <TooltipSection
          key="script"
          title={t("overviewStep.scriptPreview", {
            language: step.script.language,
          })}
        >
          <div className={tooltipStyles.valueCode}>
            {preview}
            {lineCount > 5 && (
              <div className={tooltipStyles.codeMore}>
                {t("overviewStep.moreLines", { count: lineCount - 5 })}
              </div>
            )}
          </div>
        </TooltipSection>
      );
    } else if (goals) {
      sections.push(
        <TooltipSection key="goals" title={t("stepFooter.flowGoals")}>
          {goals}
        </TooltipSection>
      );
    } else if (step.http) {
      const method = step.http.invoke?.method || DEFAULT_HTTP_METHOD;
      sections.push(
        <TooltipSection key="method" title={t("overviewStep.httpMethod")}>
          {method}
        </TooltipSection>
      );
      sections.push(
        <TooltipSection
          key="endpoint"
          title={t("overviewStep.endpointUrl")}
          icon={<IconEndpoint />}
          truncate
        >
          {step.http.invoke?.endpoint}
        </TooltipSection>
      );

      if (step.http.compensate) {
        const compensateMethod =
          step.http.compensate.method || DEFAULT_HTTP_METHOD;
        sections.push(
          <TooltipSection
            key="compensate"
            title={t("overviewStep.compensateUrl")}
            icon={<IconCompensate />}
            truncate
          >
            {`${compensateMethod} ${step.http.compensate.endpoint}`}
          </TooltipSection>
        );
      }

      if (step.http.health) {
        sections.push(
          <TooltipSection
            key="health-check"
            title={t("overviewStep.healthCheckUrl")}
            icon={<IconHealthCheck />}
            truncate
          >
            {`GET ${step.http.health}`}
          </TooltipSection>
        );
      }
    }

    sections.push(
      <TooltipSection
        key="health"
        title={t("overviewStep.healthStatus")}
        icon={<HealthDot status={healthStatus} />}
      >
        {healthText}
      </TooltipSection>
    );

    if (showNodeHealth) {
      sections.push(
        <TooltipSection key="nodes" title={t("overviewStep.nodeHealth")}>
          <div className={styles.nodeHealthList}>
            {healthNodes.map((nodeHealth) => (
              <span
                key={nodeHealth.nodeId}
                className={`${styles.nodeBubble} ${
                  styles[`nodeBubble${nodeHealth.status}`]
                }`}
                title={nodeHealthText(t, nodeHealth.status, nodeHealth.error)}
              >
                {nodeHealth.nodeId}
              </span>
            ))}
          </div>
        </TooltipSection>
      );
    }

    return { displayInfo, tooltipSections: sections };
  }, [step, healthStatus, healthNodes, healthText, showNodeHealth, t]);

  return (
    <Tooltip
      trigger={
        <div className={`${styles.footer} step-footer`}>
          <StepInfoDisplay displayInfo={displayInfo} />
          <div className={styles.actions}>
            <div className={styles.healthStatus}>
              <HealthDot status={healthIconClass as HealthStatus} />
            </div>
          </div>
        </div>
      }
    >
      <>{tooltipSections}</>
    </Tooltip>
  );
};

export default Footer;
