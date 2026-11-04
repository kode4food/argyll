import React from "react";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import useDropdown from "@/app/hooks/useDropdown";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import sharedFormStyles from "@/app/styles/components/form.module.css";
import formStyles from "./StepEditorForm.module.css";

export interface InlineSelectOption {
  value: string;
  label: string;
  icon?: React.ReactNode;
  disabled?: boolean;
  highlight?: boolean;
}

interface InlineSelectDropdownProps {
  ariaLabel?: string;
  className?: string;
  disabled?: boolean;
  faceIcon?: React.ReactNode;
  onChange: (value: string) => void;
  options: InlineSelectOption[];
  placeholder?: string;
  value: string;
}

const InlineSelectDropdown: React.FC<InlineSelectDropdownProps> = ({
  ariaLabel,
  className,
  disabled,
  faceIcon,
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
  const hasIconFace = faceIcon !== undefined;

  return (
    <div
      ref={wrapperRef}
      className={[
        hasIconFace
          ? formStyles.iconDropdownWrapper
          : formStyles.inlineSelectWrapper,
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      onKeyDown={disabled ? undefined : handleKeyDown}
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={[
          hasIconFace
            ? formStyles.iconDropdownFace
            : formStyles.inlineSelectButton,
          open &&
            (hasIconFace
              ? formStyles.iconDropdownFaceOpen
              : formStyles.inlineSelectButtonOpen),
          disabled && sharedFormStyles.disabledField,
        ]
          .filter(Boolean)
          .join(" ")}
        aria-label={ariaLabel}
        aria-expanded={open}
        aria-haspopup="listbox"
        title={hasIconFace ? selected?.label : undefined}
        disabled={disabled}
      >
        {hasIconFace ? faceIcon : (selected?.label ?? placeholder ?? value)}
      </button>
      {open && !disabled && (
        <div
          className={dropdownStyles.list}
          role="listbox"
          aria-label={hasIconFace ? ariaLabel : undefined}
          data-ui-overlay="dropdown"
        >
          {options.map((opt, index) => (
            <DropdownOption
              key={opt.value}
              icon={opt.icon}
              isActive={opt.value === value}
              isDisabled={opt.disabled}
              isEmphasized={opt.highlight}
              isHighlighted={index === highlightedIndex}
              onHover={() => setHighlightedIndex(index)}
              onSelect={() => {
                onChange(opt.value);
                setOpen(false);
              }}
            >
              {opt.label}
            </DropdownOption>
          ))}
        </div>
      )}
    </div>
  );
};

export default InlineSelectDropdown;
