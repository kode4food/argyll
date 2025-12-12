import React from "react";
import styles from "./Spinner.module.css";

interface SpinnerProps {
  size?: "sm" | "md" | "lg";
  color?: "primary" | "white";
}

const Spinner: React.FC<SpinnerProps> = ({
  size = "md",
  color = "primary",
}) => {
  const sizeClass =
    styles[`spinner${size.charAt(0).toUpperCase() + size.slice(1)}`];
  const colorClass =
    styles[`spinner${color.charAt(0).toUpperCase() + color.slice(1)}`];

  return <div className={`${styles.spinner} ${sizeClass} ${colorClass}`} />;
};

export default Spinner;
