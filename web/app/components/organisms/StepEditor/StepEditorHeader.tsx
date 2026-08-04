import React from "react";
import { useT } from "@/app/i18n";
import { IconMemoizable } from "@/utils/iconRegistry";
import IconCheckbox from "@/app/components/molecules/IconCheckbox";
import styles from "./StepEditor.module.css";

interface StepEditorHeaderProps {
  isCreateMode: boolean;
  stepId: string;
  memoizable: boolean;
  memoizableDisabled: boolean;
  onMemoizableChange: (value: boolean) => void;
}

const StepEditorHeader: React.FC<StepEditorHeaderProps> = ({
  isCreateMode,
  stepId,
  memoizable,
  memoizableDisabled,
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
        <IconCheckbox
          checked={memoizable}
          Icon={IconMemoizable}
          label={t("stepEditor.memoizableLabel")}
          onChange={onMemoizableChange}
          disabled={memoizableDisabled}
          title={
            memoizableDisabled
              ? t("stepEditor.memoizableDisabled")
              : t("stepEditor.memoizableTitle")
          }
        />
      </div>
    </div>
  );
};

export default StepEditorHeader;
