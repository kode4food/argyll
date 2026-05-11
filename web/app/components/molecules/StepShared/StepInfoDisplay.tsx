import React from "react";
import styles from "./StepFooter.module.css";

export type DisplayInfo = {
  icon: React.ComponentType<{ className?: string }>;
  text: string;
  className?: string;
} | null;

interface StepInfoDisplayProps {
  displayInfo: DisplayInfo;
}

const StepInfoDisplay: React.FC<StepInfoDisplayProps> = ({ displayInfo }) => {
  if (!displayInfo) return null;
  return (
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
  );
};

export default StepInfoDisplay;
