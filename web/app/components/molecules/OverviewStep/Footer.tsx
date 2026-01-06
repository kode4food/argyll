import React, { useMemo } from "react";
import { Step, HealthStatus } from "@/app/api";
import { getHealthIconClass, getHealthStatusText } from "@/utils/healthUtils";
import Tooltip from "@/app/components/atoms/Tooltip";
import HealthDot from "@/app/components/atoms/HealthDot";
import TooltipSection from "@/app/components/atoms/TooltipSection";
import styles from "../StepShared/StepFooter.module.css";
import {
  formatScriptPreview,
  getScriptIcon,
  getHttpIcon,
  formatScriptForTooltip,
} from "@/utils/stepFooterUtils";
import tooltipStyles from "@/app/components/atoms/TooltipSection/TooltipSection.module.css";

interface FooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
}

const Footer: React.FC<FooterProps> = ({ step, healthStatus, healthError }) => {
  const healthIconClass = getHealthIconClass(healthStatus, step.type);
  const healthText = getHealthStatusText(healthStatus, healthError);

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

    return { displayInfo, tooltipSections: sections };
  }, [step, healthStatus, healthText]);

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
