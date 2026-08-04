import React from "react";
import { LucideIcon } from "lucide-react";
import styles from "./IconCheckbox.module.css";

interface IconCheckboxProps {
  checked: boolean;
  Icon: LucideIcon;
  label: string;
  onChange: (value: boolean) => void;
  disabled?: boolean;
  title?: string;
}

const IconCheckbox: React.FC<IconCheckboxProps> = ({
  checked,
  Icon,
  label,
  onChange,
  disabled,
  title,
}) => (
  <label className={styles.label} title={title}>
    <span className={styles.icon}>
      <Icon aria-hidden="true" />
    </span>
    <span>{label}</span>
    <input
      type="checkbox"
      checked={checked}
      onChange={(e) => onChange(e.target.checked)}
      disabled={disabled}
      className={styles.checkbox}
    />
  </label>
);

export default IconCheckbox;
