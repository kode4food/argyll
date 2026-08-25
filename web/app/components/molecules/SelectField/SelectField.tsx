import React from "react";
import useDropdown from "@/app/hooks/useDropdown";
import { type LucideIcon } from "@/utils/iconRegistry";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./SelectField.module.css";

export interface SelectFieldOption {
  disabled?: boolean;
  Icon?: LucideIcon;
  label: string;
  title?: string;
  value: string;
}

export interface SelectFieldProps {
  ariaLabel?: string;
  className?: string;
  label?: string;
  onChange: (value: string) => void;
  options: SelectFieldOption[];
  value: string;
}

const SelectField: React.FC<SelectFieldProps> = ({
  ariaLabel,
  className,
  label,
  onChange,
  options,
  value,
}) => {
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown(options, value, onChange);
  const selected = options.find((option) => option.value === value);
  const SelectedIcon = selected?.Icon;

  return (
    <div className={[styles.field, className].filter(Boolean).join(" ")}>
      {label && <label className={styles.label}>{label}</label>}
      <div
        ref={wrapperRef}
        className={styles.wrapper}
        onKeyDown={handleKeyDown}
      >
        <button
          type="button"
          onClick={() => setOpen((isOpen) => !isOpen)}
          className={`${styles.face} ${open ? styles.faceOpen : ""}`}
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
        >
          {SelectedIcon && <SelectedIcon className={styles.icon} />}
          <span>{selected?.label ?? value}</span>
        </button>
        {open && (
          <div
            className={`${dropdownStyles.list} ${styles.list}`}
            role="listbox"
            data-ui-overlay="dropdown"
          >
            {options.map((option, index) => (
              <button
                key={option.value}
                type="button"
                role="option"
                aria-selected={value === option.value}
                title={option.title ?? option.label}
                disabled={option.disabled}
                className={`${dropdownStyles.item} ${
                  value === option.value ? dropdownStyles.itemActive : ""
                } ${option.disabled ? dropdownStyles.itemDisabled : ""} ${
                  index === highlightedIndex
                    ? dropdownStyles.itemHighlighted
                    : ""
                }`}
                onMouseEnter={() => setHighlightedIndex(index)}
                onClick={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
              >
                {option.Icon && (
                  <span className={dropdownStyles.itemIcon}>
                    <option.Icon className={styles.icon} />
                  </span>
                )}
                <span className={dropdownStyles.itemLabel}>{option.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default SelectField;
