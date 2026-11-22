import React from "react";
import { SCRIPT_LANGUAGE_ARGYLL, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import useDropdown from "@/app/hooks/useDropdown";
import { useT } from "@/app/i18n";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import DropdownOption from "@/app/components/atoms/DropdownOption";
import ScriptLanguageLogo from "@/app/components/atoms/ScriptLanguageLogo";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import styles from "./ScriptLanguageInlineInput.module.css";

const SCRIPT_LANGUAGE_OPTIONS = [
  { value: SCRIPT_LANGUAGE_ARGYLL, labelKey: "script.language.argyll" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
];

export interface ScriptLanguageInlineInputProps {
  ariaLabel: string;
  className?: string;
  language?: string;
  leading?: React.ReactNode;
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
  leading,
  onLanguageChange,
  onScriptChange,
  placeholder,
  script,
  title,
}) => {
  const t = useT();
  const activeLanguage = language || SCRIPT_LANGUAGE_ARGYLL;
  const {
    open,
    setOpen,
    highlightedIndex,
    setHighlightedIndex,
    wrapperRef,
    handleKeyDown,
    selectValue,
  } = useDropdown(SCRIPT_LANGUAGE_OPTIONS, activeLanguage, onLanguageChange);

  const activeOption = SCRIPT_LANGUAGE_OPTIONS.find(
    (o) => o.value === activeLanguage
  );

  return (
    <SegmentedGroup className={className}>
      {leading}
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
          <ScriptLanguageLogo
            language={activeLanguage}
            name={activeOption ? t(activeOption.labelKey) : activeLanguage}
          />
        </button>
        {open && (
          <div
            className={dropdownStyles.list}
            role="listbox"
            data-ui-overlay="dropdown"
          >
            {SCRIPT_LANGUAGE_OPTIONS.map((opt, index) => (
              <DropdownOption
                key={opt.value}
                isActive={opt.value === activeLanguage}
                isHighlighted={index === highlightedIndex}
                onHover={() => setHighlightedIndex(index)}
                onSelect={() => selectValue(opt.value)}
              >
                <ScriptLanguageLogo
                  language={opt.value}
                  name={t(opt.labelKey)}
                  showName
                />
              </DropdownOption>
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
