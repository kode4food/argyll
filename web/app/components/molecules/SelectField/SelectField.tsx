import React from "react";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
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

  const listRef = React.useRef<HTMLDivElement>(null);
  useAnchoredList({
    isOpen: open,
    anchorRef: wrapperRef,
    listRef,
    shouldMatchWidth: true,
  });

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
                onSelect={() => {
                  onChange(option.value);
                  setOpen(false);
                }}
              >
                {option.label}
              </DropdownOption>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default SelectField;
