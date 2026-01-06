import React from "react";
import { Server } from "lucide-react";
import styles from "./EmptyState.module.css";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
  className?: string;
  iconClassName?: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({
  icon = <Server className={styles.icon} />,
  title,
  description,
  action,
  className = "",
  iconClassName = "",
}) => {
  const renderedIcon =
    icon && React.isValidElement<{ className?: string }>(icon)
      ? React.cloneElement(icon, {
          className: [styles.icon, iconClassName, icon.props.className]
            .filter(Boolean)
            .join(" "),
        })
      : icon;

  return (
    <div className={`${styles.container} ${className}`}>
      {renderedIcon}
      <h3 className={styles.title}>{title}</h3>
      <p className={styles.description}>{description}</p>
      {action && <div className={styles.action}>{action}</div>}
    </div>
  );
};

export default EmptyState;
