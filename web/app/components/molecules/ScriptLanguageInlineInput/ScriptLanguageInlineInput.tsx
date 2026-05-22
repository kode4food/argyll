import React from "react";
import {
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";
import useDropdown from "@/app/hooks/useDropdown";
import { useT } from "@/app/i18n";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import styles from "./ScriptLanguageInlineInput.module.css";

const SCRIPT_LANGUAGE_OPTIONS = [
  { value: SCRIPT_LANGUAGE_JPATH, labelKey: "script.language.jpath" },
  { value: SCRIPT_LANGUAGE_ALE, labelKey: "script.language.ale" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
];

export interface ScriptLanguageInlineInputProps {
  ariaLabel: string;
  className?: string;
  language?: string;
  onLanguageChange: (language: string) => void;
  onScriptChange: (script: string) => void;
  placeholder: string;
  script: string;
  title?: string;
}

const ScriptLanguageInlineInput: React.FC<ScriptLanguageInlineInputProps> = ({
  ariaLabel,
  className,
  language,
  onLanguageChange,
  onScriptChange,
  placeholder,
  script,
  title,
}) => {
  const t = useT();
  const activeLanguage = language || SCRIPT_LANGUAGE_JPATH;
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
  } = useDropdown(SCRIPT_LANGUAGE_OPTIONS, activeLanguage, onLanguageChange);

  const activeOption = SCRIPT_LANGUAGE_OPTIONS.find(
    (o) => o.value === activeLanguage
  );

  return (
    <SegmentedGroup className={className}>
      <div
        ref={wrapperRef}
        className={styles.languageWrapper}
        onKeyDown={handleKeyDown}
      >
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          className={`${styles.languageButton} ${open ? styles.languageButtonOpen : ""}`}
          aria-label={ariaLabel}
          aria-expanded={open}
          aria-haspopup="listbox"
        >
          {activeOption ? t(activeOption.labelKey) : activeLanguage}
        </button>
        {open && (
          <div
            className={dropdownStyles.list}
            role="listbox"
            data-ui-overlay="dropdown"
          >
            {SCRIPT_LANGUAGE_OPTIONS.map((opt, index) => (
              <button
                key={opt.value}
                type="button"
                role="option"
                aria-selected={opt.value === activeLanguage}
                className={`${dropdownStyles.item} ${opt.value === activeLanguage ? dropdownStyles.itemActive : ""} ${index === highlightedIndex ? dropdownStyles.itemHighlighted : ""}`}
                onMouseEnter={() => setHighlightedIndex(index)}
                onClick={() => {
                  onLanguageChange(opt.value);
                  setOpen(false);
                }}
              >
                <span className={dropdownStyles.itemLabel}>
                  {t(opt.labelKey)}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
      <input
        type="text"
        value={script}
        onChange={(e) => onScriptChange(e.target.value)}
        placeholder={placeholder}
        className={styles.input}
        title={title}
      />
    </SegmentedGroup>
  );
};

export default ScriptLanguageInlineInput;
