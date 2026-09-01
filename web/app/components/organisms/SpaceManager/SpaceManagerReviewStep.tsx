import React from "react";
import { SCRIPT_LANGUAGE_LUA, ScriptConfig } from "@/app/api";
import ScriptConfigEditor from "@/app/components/organisms/StepEditor/ScriptConfigEditor";
import { predicateLanguageOptions } from "@/app/components/organisms/StepEditor/stepEditorConstants";
import { useT } from "@/app/i18n";
import { IconPredicate } from "@/utils/iconRegistry";
import styles from "./SpaceManager.module.css";

interface SpaceManagerReviewStepProps {
  isScriptMode: boolean;
  onEditScript: () => void;
  onLanguageChange: (language: string) => void;
  onScriptChange: (script: string) => void;
  selector?: ScriptConfig;
}

const SpaceManagerReviewStep: React.FC<SpaceManagerReviewStepProps> = ({
  isScriptMode,
  onEditScript,
  onLanguageChange,
  onScriptChange,
  selector,
}) => {
  const t = useT();
  return (
    <div className={styles.detail}>
      <ScriptConfigEditor
        Icon={IconPredicate}
        label={t("spaceManager.selectorLabel")}
        value={selector?.script ?? ""}
        onChange={onScriptChange}
        language={selector?.language ?? SCRIPT_LANGUAGE_LUA}
        onLanguageChange={onLanguageChange}
        languageOptions={predicateLanguageOptions}
        readOnly={!isScriptMode}
      />
      {!isScriptMode && (
        <button
          type="button"
          className={styles.advanced}
          onClick={onEditScript}
        >
          {t("spaceManager.editScript")}
        </button>
      )}
    </div>
  );
};

export default SpaceManagerReviewStep;
