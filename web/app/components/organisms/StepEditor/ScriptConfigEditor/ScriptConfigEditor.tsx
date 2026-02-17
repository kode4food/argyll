import React from "react";
import { SCRIPT_LANGUAGE_ALE, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import formStyles from "../StepEditorForm.module.css";
import { useT } from "@/app/i18n";

interface ScriptLanguageOption {
  value: string;
  labelKey: string;
}

interface ScriptConfigEditorProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  language: string;
  onLanguageChange: (language: string) => void;
  languageOptions?: ScriptLanguageOption[];
  readOnly?: boolean;
  containerClassName?: string;
}

const defaultLanguageOptions: ScriptLanguageOption[] = [
  { value: SCRIPT_LANGUAGE_ALE, labelKey: "script.language.ale" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
];

/**
 * Unified editor for script configurations (both step scripts and predicates).
 * Combines language selector and code editor into a single reusable component.
 */
const ScriptConfigEditor: React.FC<ScriptConfigEditorProps> = ({
  label,
  value,
  onChange,
  language,
  onLanguageChange,
  languageOptions = defaultLanguageOptions,
  readOnly = false,
  containerClassName = formStyles.scriptEditorContainer,
}) => {
  const t = useT();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>{label}</label>
        {!readOnly && (
          <div className={formStyles.languageSelectorGroup}>
            {languageOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={(e) => {
                  onLanguageChange(option.value);
                  e.currentTarget.blur();
                }}
                className={`${formStyles.languageButton} ${language === option.value ? formStyles.languageButtonActive : ""}`}
                title={t(option.labelKey)}
              >
                {t(option.labelKey)}
              </button>
            ))}
          </div>
        )}
      </div>
      <div className={containerClassName}>
        <ScriptEditor
          value={value}
          onChange={onChange}
          language={language}
          readOnly={readOnly}
        />
      </div>
    </div>
  );
};

export default ScriptConfigEditor;
