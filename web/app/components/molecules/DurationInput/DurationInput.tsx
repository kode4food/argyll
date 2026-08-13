import React from "react";
import { IconDuration } from "@/utils/iconRegistry";
import { useDurationInput } from "./useDurationInput";
import styles from "./DurationInput.module.css";
import { useT } from "@/app/i18n";

interface DurationInputProps {
  value: number; // milliseconds
  onChange: (milliseconds: number) => void;
  className?: string;
  placeholderMs?: number;
}

const DurationInput: React.FC<DurationInputProps> = ({
  value,
  onChange,
  className,
  placeholderMs,
}) => {
  const t = useT();
  const { inputValue, isValid, placeholder, handlers } = useDurationInput(
    value,
    onChange,
    placeholderMs
  );

  return (
    <div className={`${styles.durationInput} ${className || ""}`}>
      <IconDuration className={styles.icon} />
      <input
        type="text"
        value={inputValue}
        onChange={handlers.onChange}
        onFocus={handlers.onFocus}
        onBlur={handlers.onBlur}
        className={`${styles.input} ${!isValid ? styles.invalid : ""}`}
        placeholder={placeholder ?? t("durationInput.placeholder")}
        title={t("durationInput.title")}
      />
    </div>
  );
};

export default DurationInput;
