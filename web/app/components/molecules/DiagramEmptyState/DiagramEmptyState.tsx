import React from "react";
import EmptyState from "@/app/components/molecules/EmptyState";
import styles from "./DiagramEmptyState.module.css";

interface DiagramEmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
  iconClassName?: string;
}

const DiagramEmptyState: React.FC<DiagramEmptyStateProps> = ({
  icon,
  title,
  description,
  action,
  iconClassName,
}) => {
  return (
    <div className={styles.wrapper}>
      <EmptyState
        icon={icon}
        title={title}
        description={description}
        action={action}
        iconClassName={iconClassName}
        className={styles.padding}
      />
    </div>
  );
};

export default DiagramEmptyState;
