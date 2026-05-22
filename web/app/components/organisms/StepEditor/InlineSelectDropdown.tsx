import React from "react";
import useDropdown from "@/app/hooks/useDropdown";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import formStyles from "./StepEditorForm.module.css";

export interface InlineSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
  highlight?: boolean;
}

interface InlineSelectDropdownProps {
  ariaLabel?: string;
  className?: string;
  disabled?: boolean;
  onChange: (value: string) => void;
  options: InlineSelectOption[];
  placeholder?: string;
  value: string;
}

const InlineSelectDropdown: React.FC<InlineSelectDropdownProps> = ({
  ariaLabel,
  className,
  disabled,
  onChange,
  options,
  placeholder,
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

  const selected = options.find((o) => o.value === value);
  const label = selected?.label ?? placeholder ?? value;

  return (
    <div
      ref={wrapperRef}
      className={[formStyles.inlineSelectWrapper, className]
        .filter(Boolean)
        .join(" ")}
      onKeyDown={disabled ? undefined : handleKeyDown}
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`${formStyles.inlineSelectButton} ${open ? formStyles.inlineSelectButtonOpen : ""} ${disabled ? formStyles.inlineSelectDisabled : ""}`}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="listbox"
        disabled={disabled}
      >
        {label}
      </button>
      {open && !disabled && (
        <div
          className={dropdownStyles.list}
          role="listbox"
          data-ui-overlay="dropdown"
        >
          {options.map((opt, index) => (
            <button
              key={opt.value}
              type="button"
              role="option"
              aria-selected={opt.value === value}
              disabled={opt.disabled}
              className={`${dropdownStyles.item} ${opt.value === value ? dropdownStyles.itemActive : ""} ${opt.highlight ? dropdownStyles.itemHighlight : ""} ${opt.disabled ? dropdownStyles.itemDisabled : ""} ${index === highlightedIndex ? dropdownStyles.itemHighlighted : ""}`}
              onMouseEnter={() => setHighlightedIndex(index)}
              onClick={() => {
                if (!opt.disabled) {
                  onChange(opt.value);
                  setOpen(false);
                }
              }}
            >
              <span className={dropdownStyles.itemLabel}>{opt.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default InlineSelectDropdown;
