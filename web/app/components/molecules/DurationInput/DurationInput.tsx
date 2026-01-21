import React from "react";
import { IconDuration } from "@/utils/iconRegistry";
import { useDurationInput } from "./useDurationInput";
import styles from "./DurationInput.module.css";
import { useT } from "@/app/i18n";

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
  const t = useT();
  const { inputValue, isValid, handlers } = useDurationInput(value, onChange);

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
        placeholder={t("durationInput.placeholder")}
        title={t("durationInput.title")}
      />
    </div>
  );
};

export default DurationInput;
