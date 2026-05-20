import React from "react";
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
  const [open, setOpen] = React.useState(false);
  const wrapperRef = React.useRef<HTMLDivElement>(null);

  const selected = options.find((o) => o.value === value);
  const label = selected?.label ?? placeholder ?? value;

  React.useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  return (
    <div
      ref={wrapperRef}
      className={[formStyles.inlineSelect, className].filter(Boolean).join(" ")}
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
        <div className={dropdownStyles.list} role="listbox">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="option"
              aria-selected={opt.value === value}
              disabled={opt.disabled}
              className={`${dropdownStyles.item} ${
                opt.value === value ? dropdownStyles.itemActive : ""
              } ${opt.highlight ? dropdownStyles.itemHighlight : ""} ${
                opt.disabled ? dropdownStyles.itemDisabled : ""
              }`}
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
