import React from "react";
import useDropdown from "@/app/hooks/useDropdown";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import formStyles from "./StepEditorForm.module.css";

export interface IconDropdownOption {
  value: string;
  label: string;
  icon: React.ReactNode;
}

interface IconDropdownProps {
  ariaLabel: string;
  faceIcon: React.ReactNode;
  onChange: (value: string) => void;
  options: IconDropdownOption[];
  value: string;
}

const IconDropdown: React.FC<IconDropdownProps> = ({
  ariaLabel,
  faceIcon,
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

  const selected = options.find((o) => o.value === value);

  return (
    <div
      ref={wrapperRef}
      className={formStyles.iconDropdownWrapper}
      onKeyDown={handleKeyDown}
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`${formStyles.iconDropdownFace} ${open ? formStyles.iconDropdownFaceOpen : ""}`}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="listbox"
        title={selected?.label}
      >
        {faceIcon}
      </button>
      {open && (
        <div
          className={dropdownStyles.list}
          role="listbox"
          aria-label={ariaLabel}
          data-ui-overlay="dropdown"
        >
          {options.map((opt, index) => (
            <button
              key={opt.value}
              type="button"
              role="option"
              aria-selected={opt.value === value}
              className={`${dropdownStyles.item} ${opt.value === value ? dropdownStyles.itemActive : ""} ${index === highlightedIndex ? dropdownStyles.itemHighlighted : ""}`}
              onMouseEnter={() => setHighlightedIndex(index)}
              onClick={() => {
                onChange(opt.value);
                setOpen(false);
              }}
            >
              <span className={dropdownStyles.itemIcon}>{opt.icon}</span>
              <span className={dropdownStyles.itemLabel}>{opt.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default IconDropdown;
