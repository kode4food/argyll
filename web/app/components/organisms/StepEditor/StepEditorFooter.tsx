import React from "react";
import { useT } from "@/app/i18n";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";

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
        <div className={formStyles.editorModeToggleGroup}>
          <button
            type="button"
            className={`${formStyles.editorModeToggle} ${
              editorMode === "basic" ? formStyles.editorModeToggleActive : ""
            }`}
            onClick={() => onEditorModeChange("basic")}
          >
            {t("stepEditor.modeBasic")}
          </button>
          <button
            type="button"
            className={`${formStyles.editorModeToggle} ${
              editorMode === "json" ? formStyles.editorModeToggleActive : ""
            }`}
            onClick={() => onEditorModeChange("json")}
          >
            {t("stepEditor.modeJson")}
          </button>
        </div>
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
