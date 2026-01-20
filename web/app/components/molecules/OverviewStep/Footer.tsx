import React, { useMemo } from "react";
import { Step, HealthStatus } from "@/app/api";
import { getHealthIconClass } from "@/utils/healthUtils";
import Tooltip from "@/app/components/atoms/Tooltip";
import HealthDot from "@/app/components/atoms/HealthDot";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "../StepShared/StepFooter.module.css";
import {
  formatScriptPreview,
  getScriptIcon,
  getHttpIcon,
  formatScriptForTooltip,
  getFlowIcon,
} from "@/utils/stepFooterUtils";
import tooltipStyles from "@/app/components/atoms/TooltipSection/TooltipSection.module.css";
import { useT } from "@/app/i18n";

interface FooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
}

const Footer: React.FC<FooterProps> = ({ step, healthStatus, healthError }) => {
  const t = useT();
  const healthIconClass = getHealthIconClass(healthStatus, step.type);
  const healthText =
    healthStatus === "healthy"
      ? t("healthStatus.healthy")
      : healthStatus === "unhealthy"
        ? healthError || t("healthStatus.unhealthy")
        : healthStatus === "unconfigured"
          ? t("healthStatus.unconfigured")
          : t("healthStatus.unknown");

  const { displayInfo, tooltipSections } = useMemo(() => {
    let displayInfo: {
      icon: React.ComponentType<any>;
      text: string;
      className?: string;
    } | null = null;

    if (step.type === "script" && step.script) {
      const ScriptIcon = getScriptIcon(step.script.language);
      const scriptPreview = formatScriptPreview(step.script.script);
      displayInfo = {
        icon: ScriptIcon,
        text: scriptPreview,
        className: "endpoint-script",
      };
    } else if (step.type === "flow" && step.flow?.goals?.length) {
      const FlowIcon = getFlowIcon();
      displayInfo = {
        icon: FlowIcon,
        text: step.flow.goals.join(", "),
      };
    } else if (step.http) {
      const HttpIcon = getHttpIcon(step.type);
      displayInfo = {
        icon: HttpIcon,
        text: step.http.endpoint,
      };
    }

    const sections: React.ReactElement[] = [];

    if (step.type === "script" && step.script) {
      const { preview, lineCount } = formatScriptForTooltip(
        step.script.script,
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
    } else if (step.type === "flow" && step.flow?.goals?.length) {
      sections.push(
        <TooltipSection key="goals" title={t("stepFooter.flowGoals")}>
          {step.flow.goals.join(", ")}
        </TooltipSection>
      );
    } else if (step.http) {
      sections.push(
        <TooltipSection key="endpoint" title={t("overviewStep.endpointUrl")}>
          {step.http.endpoint}
        </TooltipSection>
      );

      if (step.http.health_check) {
        sections.push(
          <TooltipSection
            key="health-check"
            title={t("overviewStep.healthCheckUrl")}
          >
            {step.http.health_check}
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

    return { displayInfo, tooltipSections: sections };
  }, [step, healthStatus, healthText, t]);

  return (
    <Tooltip
      trigger={
        <div className={`${styles.footer} step-footer`}>
          {displayInfo && (
            <div className={styles.infoDisplay}>
              {React.createElement(displayInfo.icon, {
                className: `step-type-icon ${styles.icon}`,
              })}
              <span
                className={`${styles.endpoint} ${
                  displayInfo.className === "endpoint-script"
                    ? styles.endpointScript
                    : ""
                } step-endpoint`}
              >
                {displayInfo.text}
              </span>
            </div>
          )}
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
