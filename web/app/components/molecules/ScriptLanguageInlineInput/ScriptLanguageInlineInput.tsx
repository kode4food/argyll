import React from "react";
import {
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";
import { useT } from "@/app/i18n";
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

  return (
    <div className={[styles.wrap, className].filter(Boolean).join(" ")}>
      <select
        value={activeLanguage}
        onChange={(e) => onLanguageChange(e.target.value)}
        className={styles.select}
        title={ariaLabel}
        aria-label={ariaLabel}
      >
        {SCRIPT_LANGUAGE_OPTIONS.map((option) => (
          <option key={option.value} value={option.value}>
            {t(option.labelKey)}
          </option>
        ))}
      </select>
      <input
        type="text"
        value={script}
        onChange={(e) => onScriptChange(e.target.value)}
        placeholder={placeholder}
        className={styles.input}
        title={title}
      />
    </div>
  );
};

export default ScriptLanguageInlineInput;
