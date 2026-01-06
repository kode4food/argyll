import React from "react";
import { Clock } from "lucide-react";
import { useDurationInput } from "./useDurationInput";
import styles from "./DurationInput.module.css";

interface DurationInputProps {
  value: number; // milliseconds
  onChange: (milliseconds: number) => void;
  className?: string;
}

const DurationInput: React.FC<DurationInputProps> = ({
  value,
  onChange,
  className,
}) => {
  const { inputValue, isValid, handlers } = useDurationInput(value, onChange);

  return (
    <div className={`${styles.durationInput} ${className || ""}`}>
      <Clock className={styles.icon} />
      <input
        type="text"
        value={inputValue}
        onChange={handlers.onChange}
        onFocus={handlers.onFocus}
        onBlur={handlers.onBlur}
        className={`${styles.input} ${!isValid ? styles.invalid : ""}`}
        placeholder="e.g. 5d, 2 days 3h, 1.5 days"
        title="Examples: 5d, 2 days, 1 day 3 hrs, 1.5 days, 2d 7h"
      />
    </div>
  );
};

export default DurationInput;
