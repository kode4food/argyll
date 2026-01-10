import React from "react";
import styles from "./DiagramView.module.css";

interface DiagramViewProps {
  children: React.ReactNode;
}

const DiagramView = React.forwardRef<HTMLDivElement, DiagramViewProps>(
  ({ children }, ref) => {
    return (
      <div className={styles.wrapper} ref={ref}>
        {children}
      </div>
    );
  }
);

DiagramView.displayName = "DiagramView";

export default DiagramView;
