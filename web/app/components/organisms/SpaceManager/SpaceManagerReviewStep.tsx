import React from "react";
import { ScriptConfig } from "@/app/api";
import ScriptConfigEditor from "@/app/components/organisms/StepEditor/ScriptConfigEditor";
import { useT } from "@/app/i18n";
import { IconPredicate } from "@/utils/iconRegistry";
import styles from "./SpaceManager.module.css";
import { useSelectorCompletions } from "./useSelectorCompletions";

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
  const { language, completions } = useSelectorCompletions(selector);
  return (
    <div className={styles.detail}>
      <ScriptConfigEditor
        Icon={IconPredicate}
        label={t("spaceManager.selectorLabel")}
        completions={completions}
        value={selector?.source ?? ""}
        onChange={onScriptChange}
        language={language}
        onLanguageChange={onLanguageChange}
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
