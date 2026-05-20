import React from "react";
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
  const [open, setOpen] = React.useState(false);
  const wrapperRef = React.useRef<HTMLDivElement>(null);

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
    <div ref={wrapperRef} className={formStyles.iconDropdownWrapper}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`${formStyles.iconDropdownFace} ${open ? formStyles.iconDropdownFaceOpen : ""}`}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        {faceIcon}
      </button>
      {open && (
        <div
          className={dropdownStyles.list}
          role="listbox"
          aria-label={ariaLabel}
        >
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              role="option"
              aria-selected={opt.value === value}
              className={`${dropdownStyles.item} ${
                opt.value === value ? dropdownStyles.itemActive : ""
              }`}
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
