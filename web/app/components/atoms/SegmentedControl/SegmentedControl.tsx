import React from "react";
import styles from "./SegmentedControl.module.css";

export interface SegmentedControlOption {
  value: string;
  label: string;
}

interface SegmentedControlProps {
  options: SegmentedControlOption[];
  value: string;
  onChange: (value: string) => void;
}

const SegmentedControl: React.FC<SegmentedControlProps> = ({
  options,
  value,
  onChange,
}) => (
  <div className={styles.track} role="group">
    {options.map((opt) => (
      <button
        key={opt.value}
        type="button"
        className={`${styles.segment} ${value === opt.value ? styles.segmentActive : ""}`}
        aria-pressed={value === opt.value}
        onClick={() => onChange(opt.value)}
      >
        {opt.label}
      </button>
    ))}
  </div>
);

export default SegmentedControl;
