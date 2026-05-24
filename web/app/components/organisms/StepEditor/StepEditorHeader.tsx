import React from "react";
import { useT } from "@/app/i18n";
import { IconMemoizable } from "@/utils/iconRegistry";
import styles from "./StepEditor.module.css";

interface StepEditorHeaderProps {
  isCreateMode: boolean;
  stepId: string;
  memoizable: boolean;
  onMemoizableChange: (value: boolean) => void;
}

const StepEditorHeader: React.FC<StepEditorHeaderProps> = ({
  isCreateMode,
  stepId,
  memoizable,
  onMemoizableChange,
}) => {
  const t = useT();
  return (
    <div className={styles.header}>
      <h2 className={styles.title}>
        {isCreateMode
          ? t("stepEditor.modalCreateTitle")
          : t("stepEditor.modalEditTitle", { id: stepId })}
      </h2>
      <div className={styles.headerControls}>
        <label
          className={styles.headerCheckboxLabel}
          title={t("stepEditor.memoizableTitle")}
        >
          <span className={styles.iconMd}>
            <IconMemoizable aria-hidden="true" />
          </span>
          <span>{t("stepEditor.memoizableLabel")}</span>
          <input
            type="checkbox"
            checked={memoizable}
            onChange={(e) => onMemoizableChange(e.target.checked)}
            className={styles.headerCheckbox}
          />
        </label>
      </div>
    </div>
  );
};

export default StepEditorHeader;
