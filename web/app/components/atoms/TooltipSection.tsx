import React from "react";
import styles from "./TooltipSection.module.css";

interface TooltipSectionProps {
  title: string;
  children: React.ReactNode;
  icon?: React.ReactNode;
  monospace?: boolean;
  bold?: boolean;
}

const TooltipSection: React.FC<TooltipSectionProps> = ({
  title,
  children,
  icon,
  monospace = false,
  bold = false,
}) => {
  let valueClass = styles.value;
  if (monospace) valueClass += ` ${styles.valueMonospace}`;
  if (bold) valueClass += ` ${styles.valueBold}`;

  return (
    <div className={styles.section}>
      <div className={styles.label}>{title}:</div>
      {icon ? (
        <div
          className={`${valueClass} ${styles.valueWithIcon} gap-2 flex items-start`}
        >
          <div className="mt-0.5 flex-shrink-0">{icon}</div>
          <span className="flex-1">{children}</span>
        </div>
      ) : (
        <div className={valueClass}>{children}</div>
      )}
    </div>
  );
};

export default TooltipSection;
