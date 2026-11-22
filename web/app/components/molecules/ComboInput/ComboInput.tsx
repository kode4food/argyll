import React from "react";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./ComboInput.module.css";
import { useComboInput } from "./useComboInput";

export interface ComboInputProps {
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
  const {
    open,
    shown,
    listRef,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleInputChange,
    handleComboKeyDown,
    handleTriggerClick,
    selectValue,
  } = useComboInput(suggestions, value, onChange);

  return (
    <div
      ref={wrapperRef}
      className={[styles.wrapper, className].filter(Boolean).join(" ")}
      onKeyDown={handleComboKeyDown}
    >
      <input
        type="text"
        value={value}
        onChange={(e) => handleInputChange(e.target.value)}
        placeholder={placeholder}
        className={styles.input}
        aria-label={ariaLabel}
        aria-autocomplete="list"
        aria-expanded={open}
      />
      <button
        type="button"
        tabIndex={-1}
        onClick={handleTriggerClick}
        disabled={suggestions.length === 0}
        className={`${styles.trigger} ${open ? styles.triggerOpen : ""}`}
        aria-label="Show suggestions"
      />
      {open && shown.length > 0 && (
        <div
          className={dropdownStyles.list}
          ref={listRef}
          role="listbox"
          data-ui-overlay="dropdown"
        >
          {shown.map((s, index) => (
            <DropdownOption
              key={s}
              isActive={s === value}
              isHighlighted={index === highlightedIndex}
              onHover={() => setHighlightedIndex(index)}
              onSelect={() => selectValue(s)}
            >
              {s}
            </DropdownOption>
          ))}
        </div>
      )}
    </div>
  );
};

export default ComboInput;
