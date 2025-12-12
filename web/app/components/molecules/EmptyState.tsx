import React from "react";
import { Server } from "lucide-react";
import styles from "./EmptyState.module.css";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
  className?: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({
  icon = <Server className={styles.icon} />,
  title,
  description,
  action,
  className = "",
}) => {
  return (
    <div className={`${styles.container} ${className}`}>
      {icon}
      <h3 className={styles.title}>{title}</h3>
      <p className={styles.description}>{description}</p>
      {action && <div className={styles.action}>{action}</div>}
    </div>
  );
};

export default EmptyState;
