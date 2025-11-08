import React from "react";
import { Server } from "lucide-react";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
  className?: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({
  icon = <Server className="text-neutral-text mx-auto mb-4 h-16 w-16" />,
  title,
  description,
  action,
  className = "",
}) => {
  return (
    <div className={`text-center ${className}`}>
      {icon}
      <h3 className="text-neutral-text mb-2 text-xl font-medium">{title}</h3>
      <p className="text-neutral-text mx-auto max-w-sm">{description}</p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
};

export default EmptyState;
