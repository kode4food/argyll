import React from "react";
import { createPortal } from "react-dom";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import { type LucideIcon } from "@/utils/iconRegistry";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./SelectField.module.css";
import { useSelectField } from "./useSelectField";

export interface SelectFieldOption {
  disabled?: boolean;
  Icon?: LucideIcon;
  label: string;
  title?: string;
  value: string;
}

export const textOptions = (names: string[]): SelectFieldOption[] =>
  names.map((name) => ({ label: name, value: name }));

export interface SelectFieldProps {
  ariaLabel?: string;
  className?: string;
  Icon?: LucideIcon;
  label?: string;
  onChange: (value: string) => void;
  options: SelectFieldOption[];
  placeholder?: string;
  value: string;
}

const SelectField: React.FC<SelectFieldProps> = ({
  ariaLabel,
  className,
  Icon,
  label,
  onChange,
  options,
  placeholder,
  value,
}) => {
  const {
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    listRef,
    handleKeyDown,
    selected,
    disabled,
    isOpen,
    selectValue,
  } = useSelectField(options, value, onChange);
  const SelectedIcon = selected?.Icon ?? Icon;

  return (
    <div className={[styles.field, className].filter(Boolean).join(" ")}>
      {label && <label className={styles.label}>{label}</label>}
      <div
        ref={wrapperRef}
        className={styles.wrapper}
        onKeyDown={disabled ? undefined : handleKeyDown}
      >
        <button
          type="button"
          disabled={disabled}
          onClick={() => setOpen((isOpen) => !isOpen)}
          className={`${styles.face} ${isOpen ? styles.faceOpen : ""}`}
          aria-expanded={isOpen}
          aria-haspopup="listbox"
          aria-label={ariaLabel}
        >
          {SelectedIcon && <SelectedIcon className={styles.icon} />}
          <span>{selected?.label ?? (value || placeholder)}</span>
        </button>
        {isOpen &&
          createPortal(
            <div
              ref={listRef}
              className={`${dropdownStyles.list} ${styles.list}`}
              role="listbox"
              data-ui-overlay="dropdown"
            >
              {options.map((option, index) => (
                <DropdownOption
                  key={option.value}
                  title={option.title ?? option.label}
                  icon={option.Icon && <option.Icon className={styles.icon} />}
                  isActive={value === option.value}
                  isDisabled={option.disabled}
                  isHighlighted={index === highlightedIndex}
                  onHover={() => setHighlightedIndex(index)}
                  onSelect={() => selectValue(option.value)}
                >
                  {option.label}
                </DropdownOption>
              ))}
            </div>,
            wrapperRef.current?.closest("dialog") ?? document.body
          )}
      </div>
    </div>
  );
};

export default SelectField;
