import React from "react";
import { useT } from "@/app/i18n";
import EditorModeToggle from "@/app/components/atoms/EditorModeToggle";
import styles from "./StepEditor.module.css";

interface StepEditorFooterProps {
  editorMode: "basic" | "json";
  onEditorModeChange: (mode: "basic" | "json") => void;
  onCancel: () => void;
  onSave: () => void;
  saving: boolean;
  isCreateMode: boolean;
}

const StepEditorFooter: React.FC<StepEditorFooterProps> = ({
  editorMode,
  onEditorModeChange,
  onCancel,
  onSave,
  saving,
  isCreateMode,
}) => {
  const t = useT();
  return (
    <div className={styles.footer}>
      <div className={styles.footerControls}>
        <EditorModeToggle
          editorMode={editorMode}
          onChange={onEditorModeChange}
          basicLabel={t("stepEditor.modeBasic")}
          jsonLabel={t("stepEditor.modeJson")}
        />
      </div>
      <div className={styles.footerButtons}>
        <button
          onClick={onCancel}
          disabled={saving}
          className={styles.buttonCancel}
        >
          {t("stepEditor.cancel")}
        </button>
        <button
          onClick={onSave}
          disabled={saving}
          className={styles.buttonSave}
        >
          {saving
            ? isCreateMode
              ? t("stepEditor.creating")
              : t("stepEditor.saving")
            : isCreateMode
              ? t("stepEditor.create")
              : t("stepEditor.save")}
        </button>
      </div>
    </div>
  );
};

export default StepEditorFooter;
