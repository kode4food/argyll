import React from "react";
import styles from "@/app/styles/components/dropdown.module.css";

export interface DropdownOptionProps {
  children: React.ReactNode;
  icon?: React.ReactNode;
  isActive?: boolean;
  isDisabled?: boolean;
  isEmphasized?: boolean;
  isHighlighted: boolean;
  onHover: () => void;
  onSelect: () => void;
  role?: "option" | "menuitem";
  title?: string;
}

const DropdownOption: React.FC<DropdownOptionProps> = ({
  children,
  icon,
  isActive = false,
  isDisabled = false,
  isEmphasized = false,
  isHighlighted,
  onHover,
  onSelect,
  role = "option",
  title,
}) => (
  <button
    type="button"
    role={role}
    aria-selected={role === "option" ? isActive : undefined}
    title={title}
    disabled={isDisabled}
    className={[
      styles.item,
      isActive && styles.itemActive,
      isEmphasized && styles.itemHighlight,
      isDisabled && styles.itemDisabled,
      isHighlighted && styles.itemHighlighted,
    ]
      .filter(Boolean)
      .join(" ")}
    onMouseEnter={onHover}
    onClick={() => {
      if (!isDisabled) onSelect();
    }}
  >
    {icon && <span className={styles.itemIcon}>{icon}</span>}
    <span className={styles.itemLabel}>{children}</span>
  </button>
);

export default DropdownOption;
