import React from "react";
import { HealthStatus } from "../../api";
import styles from "./HealthDot.module.css";

interface HealthDotProps {
  status: HealthStatus;
}

const HealthDot: React.FC<HealthDotProps> = ({ status }) => (
  <div className={`${styles.dot} ${styles[status]}`} />
);

export default HealthDot;
