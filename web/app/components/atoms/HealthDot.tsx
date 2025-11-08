import React from "react";
import styles from "./HealthDot.module.css";

interface HealthDotProps {
  className: string;
}

const HealthDot: React.FC<HealthDotProps> = ({ className }) => (
  <div
    className={`${styles.dot} ${styles[className as keyof typeof styles]}`}
  />
);

export default HealthDot;
