import React from "react";
import SegmentedControl from "@/app/components/atoms/SegmentedControl";
import ScriptLanguageLogo from "@/app/components/atoms/ScriptLanguageLogo";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import { type LucideIcon } from "@/utils/iconRegistry";
import formStyles from "../StepEditorForm.module.css";
import { SCRIPT_LANGUAGE_OPTIONS } from "@/app/components/organisms/StepEditor/stepEditorConstants";
import { useT } from "@/app/i18n";

interface ScriptConfigEditorProps {
  Icon: LucideIcon;
  label: string;
  value: string;
  onChange: (value: string) => void;
  completions?: readonly string[];
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
  Icon,
  label,
  value,
  onChange,
  completions,
  language,
  onLanguageChange,
  readOnly = false,
  containerClassName = formStyles.scriptEditorContainer,
}) => {
  const t = useT();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.labelWithIcon}>
          <span className={formStyles.labelIcon}>
            <Icon aria-hidden="true" />
          </span>
          {label}
        </label>
        {!readOnly && (
          <SegmentedControl
            options={SCRIPT_LANGUAGE_OPTIONS.map((opt) => ({
              value: opt.value,
              label: (
                <ScriptLanguageLogo
                  language={opt.value}
                  name={t(opt.labelKey)}
                  showName
                />
              ),
            }))}
            value={language}
            onChange={onLanguageChange}
          />
        )}
      </div>
      <div className={containerClassName}>
        <ScriptEditor
          value={value}
          onChange={onChange}
          completions={completions}
          language={language}
          readOnly={readOnly}
        />
      </div>
    </div>
  );
};

export default ScriptConfigEditor;
