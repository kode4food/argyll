import React, { useState, useEffect } from "react";
import ms from "ms";
import type { StringValue } from "ms";
import { Clock } from "lucide-react";
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
  const [inputValue, setInputValue] = useState("");
  const [isValid, setIsValid] = useState(true);
  const [isFocused, setIsFocused] = useState(false);

  useEffect(() => {
    if (!isFocused) {
      if (value) {
        setInputValue(ms(value, { long: true }));
      } else {
        setInputValue("");
      }
      setIsValid(true);
    }
  }, [value, isFocused]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value;
    setInputValue(input);

    if (!input.trim()) {
      setIsValid(true);
      onChange(0);
      return;
    }

    try {
      const parsed = ms(input as StringValue);
      if (parsed >= 0) {
        setIsValid(true);
        onChange(parsed);
      } else {
        setIsValid(false);
      }
    } catch (err) {
      setIsValid(false);
    }
  };

  const handleFocus = () => {
    setIsFocused(true);
  };

  const handleBlur = () => {
    setIsFocused(false);
    if (isValid && inputValue.trim()) {
      const parsed = ms(inputValue as StringValue);
      if (parsed >= 0) {
        setInputValue(ms(parsed, { long: true }));
      }
    }
  };

  return (
    <div className={`${styles.durationInput} ${className || ""}`}>
      <Clock size={14} className={styles.icon} />
      <input
        type="text"
        value={inputValue}
        onChange={handleChange}
        onFocus={handleFocus}
        onBlur={handleBlur}
        className={`${styles.input} ${!isValid ? styles.invalid : ""}`}
        placeholder="e.g. 5d, 2 days 3h, 1.5 days"
        title="Examples: 5d, 2 days, 1 day 3 hrs, 1.5 days, 2d 7h"
      />
    </div>
  );
};

export default DurationInput;
