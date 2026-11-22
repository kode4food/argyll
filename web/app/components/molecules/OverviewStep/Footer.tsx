import React from "react";
import { Step, HealthStatus, NodeStepHealth } from "@/app/api";
import Tooltip from "@/app/components/atoms/Tooltip";
import HealthDot from "@/app/components/atoms/HealthDot";
import styles from "@/app/components/molecules/StepShared/StepFooter.module.css";
import StepInfoDisplay from "@/app/components/molecules/StepShared/StepInfoDisplay";
import { useFooter } from "./useFooter";

interface FooterProps {
  step: Step;
  healthStatus: HealthStatus;
  healthError?: string;
  healthNodes?: NodeStepHealth[];
}

const Footer: React.FC<FooterProps> = ({ step, ...health }) => {
  const { healthIconClass, displayInfo, tooltipSections } = useFooter(
    step,
    health
  );

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
