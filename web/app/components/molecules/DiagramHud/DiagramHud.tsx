import React from "react";
import styles from "./DiagramHud.module.css";

interface DiagramHudProps {
  className?: string;
  sections: React.ReactNode[];
}

interface DiagramHudTextProps {
  children: React.ReactNode;
  nowrap?: boolean;
}

const DiagramHudSeparator: React.FC = () => (
  <span className={styles.separator} aria-hidden="true">
    |
  </span>
);

export const DiagramHudText: React.FC<DiagramHudTextProps> = ({
  children,
  nowrap = false,
}) => (
  <span
    className={[styles.text, nowrap ? styles.nowrap : ""]
      .filter(Boolean)
      .join(" ")}
  >
    {children}
  </span>
);

const DiagramHud: React.FC<DiagramHudProps> = ({ className, sections }) => (
  <div className={[styles.root, className].filter(Boolean).join(" ")}>
    <div className={styles.content}>
      {sections.map((section, idx) => (
        <React.Fragment key={idx}>
          <div className={styles.section}>{section}</div>
          {idx < sections.length - 1 && <DiagramHudSeparator />}
        </React.Fragment>
      ))}
    </div>
  </div>
);

export default DiagramHud;
