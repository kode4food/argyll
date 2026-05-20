import React from "react";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import formStyles from "./StepEditorForm.module.css";

interface ComboInputProps {
  ariaLabel?: string;
  className?: string;
  onChange: (value: string) => void;
  placeholder?: string;
  suggestions: readonly string[];
  value: string;
}

const ComboInput: React.FC<ComboInputProps> = ({
  ariaLabel,
  className,
  onChange,
  placeholder,
  suggestions,
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
    <div
      ref={wrapperRef}
      className={[formStyles.comboWrapper, className].filter(Boolean).join(" ")}
    >
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={formStyles.comboInput}
        aria-label={ariaLabel}
        aria-autocomplete="list"
        aria-expanded={open}
      />
      <button
        type="button"
        tabIndex={-1}
        onClick={() => setOpen((o) => !o)}
        className={`${formStyles.comboTrigger} ${open ? formStyles.comboTriggerOpen : ""}`}
        aria-label="Show suggestions"
      />
      {open && (
        <div className={dropdownStyles.list} role="listbox">
          {suggestions.map((s) => (
            <button
              key={s}
              type="button"
              role="option"
              aria-selected={s === value}
              className={`${dropdownStyles.item} ${
                s === value ? dropdownStyles.itemActive : ""
              }`}
              onClick={() => {
                onChange(s);
                setOpen(false);
              }}
            >
              <span className={dropdownStyles.itemLabel}>{s}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default ComboInput;
