import React from "react";
import styles from "./DiagramLayout.module.css";

interface DiagramLayoutProps {
  header?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  containerRef?: React.Ref<HTMLDivElement>;
}

const DiagramLayout: React.FC<DiagramLayoutProps> = ({
  header,
  children,
  className,
  containerRef,
}) => {
  const containerClassName = [styles.container, className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={containerClassName}>
      {header && <div className={styles.header}>{header}</div>}
      <div className={styles.diagramContainer} ref={containerRef}>
        <div className={styles.diagramContent}>{children}</div>
      </div>
    </div>
  );
};

export default DiagramLayout;
