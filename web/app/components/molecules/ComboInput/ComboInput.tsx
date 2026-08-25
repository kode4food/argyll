import React from "react";
import useDropdown from "@/app/hooks/useDropdown";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import styles from "./ComboInput.module.css";

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
  const options = suggestions.map((s) => ({ value: s }));
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown(options, value, onChange);

  const handleComboKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === " ") return;
    handleKeyDown(e);
  };

  // Fixed against the viewport so a scrolling ancestor cannot clip it
  const [anchor, setAnchor] = React.useState<DOMRect | null>(null);
  React.useLayoutEffect(() => {
    setAnchor(
      open ? (wrapperRef.current?.getBoundingClientRect() ?? null) : null
    );
  }, [open, wrapperRef]);

  return (
    <div
      ref={wrapperRef}
      className={[styles.wrapper, className].filter(Boolean).join(" ")}
      onKeyDown={handleComboKeyDown}
    >
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={styles.input}
        aria-label={ariaLabel}
        aria-autocomplete="list"
        aria-expanded={open}
      />
      <button
        type="button"
        tabIndex={-1}
        onClick={() => setOpen((o) => !o)}
        disabled={suggestions.length === 0}
        className={`${styles.trigger} ${open ? styles.triggerOpen : ""}`}
        aria-label="Show suggestions"
      />
      {open && suggestions.length > 0 && (
        <div
          className={dropdownStyles.list}
          role="listbox"
          data-ui-overlay="dropdown"
          style={
            anchor
              ? {
                  position: "fixed",
                  top: anchor.bottom + 4,
                  left: anchor.left,
                  minWidth: anchor.width,
                }
              : undefined
          }
        >
          {suggestions.map((s, index) => (
            <button
              key={s}
              type="button"
              role="option"
              aria-selected={s === value}
              className={`${dropdownStyles.item} ${s === value ? dropdownStyles.itemActive : ""} ${index === highlightedIndex ? dropdownStyles.itemHighlighted : ""}`}
              onMouseEnter={() => setHighlightedIndex(index)}
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
