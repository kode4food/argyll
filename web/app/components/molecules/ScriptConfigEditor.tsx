"use client";

import React from "react";
import { SCRIPT_LANGUAGE_ALE, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import ScriptEditor from "./ScriptEditor";
import formStyles from "../organisms/StepEditorForm.module.css";

interface ScriptConfigEditorProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  language: string;
  onLanguageChange: (language: string) => void;
  readOnly?: boolean;
  containerClassName?: string;
}

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
  readOnly = false,
  containerClassName = formStyles.scriptEditorContainer,
}) => {
  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>{label}</label>
        {!readOnly && (
          <div className={formStyles.languageSelectorGroup}>
            <button
              type="button"
              onClick={(e) => {
                onLanguageChange(SCRIPT_LANGUAGE_ALE);
                e.currentTarget.blur();
              }}
              className={`${formStyles.languageButton} ${language === SCRIPT_LANGUAGE_ALE ? formStyles.languageButtonActive : ""}`}
              title="Ale"
            >
              Ale
            </button>
            <button
              type="button"
              onClick={(e) => {
                onLanguageChange(SCRIPT_LANGUAGE_LUA);
                e.currentTarget.blur();
              }}
              className={`${formStyles.languageButton} ${language === SCRIPT_LANGUAGE_LUA ? formStyles.languageButtonActive : ""}`}
              title="Lua"
            >
              Lua
            </button>
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
