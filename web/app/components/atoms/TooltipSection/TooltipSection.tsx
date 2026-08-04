import React from "react";
import styles from "./TooltipSection.module.css";

interface TooltipSectionProps {
  title: string;
  children: React.ReactNode;
  icon?: React.ReactNode;
  monospace?: boolean;
  bold?: boolean;
  // Ellipsizes the value; only applies alongside an icon
  truncate?: boolean;
}

const TooltipSection: React.FC<TooltipSectionProps> = ({
  title,
  children,
  icon,
  monospace = false,
  bold = false,
  truncate = false,
}) => {
  let valueClass = styles.value;
  if (monospace) valueClass += ` ${styles.valueMonospace}`;
  if (bold) valueClass += ` ${styles.valueBold}`;

  const textClass = [styles.textContent, truncate && styles.truncate]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={styles.section}>
      <div className={styles.label}>{title}:</div>
      {icon ? (
        <div className={`${valueClass} ${styles.valueWithIcon}`}>
          <div className={styles.iconWrapper}>{icon}</div>
          <span className={textClass}>{children}</span>
        </div>
      ) : (
        <div className={valueClass}>{children}</div>
      )}
    </div>
  );
};

export default TooltipSection;
