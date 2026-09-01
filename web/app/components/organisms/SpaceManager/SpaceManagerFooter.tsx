import React from "react";
import { useT } from "@/app/i18n";
import { REVIEW_STEP } from "./spaceManagerUtils";
import { SpaceEditor } from "./useSpaceEditor";
import styles from "./SpaceManager.module.css";

interface SpaceManagerFooterProps {
  editor: SpaceEditor;
}

const SpaceManagerFooter: React.FC<SpaceManagerFooterProps> = ({ editor }) => {
  const t = useT();
  const {
    canSave,
    currentPreview,
    draft,
    editingId,
    editorStep,
    firstStep,
    handleBack,
    handleClose,
    handleDelete,
    handleNext,
    handleSave,
    hasValidDetails,
    hasValidQBE,
    isEditing,
    openEditor,
    saving,
  } = editor;

  if (!isEditing) {
    return (
      <>
        <button
          autoFocus
          type="button"
          className={styles.buttonPrimary}
          onClick={() => openEditor()}
        >
          {t("spaceManager.new")}
        </button>
        <button type="button" className={styles.button} onClick={handleClose}>
          {t("spaceManager.close")}
        </button>
      </>
    );
  }

  const isPreviewPending = !draft.scriptMode && !currentPreview;
  const isNextDisabled =
    saving ||
    (editorStep === 0 ? !hasValidQBE : !hasValidDetails || isPreviewPending);

  return (
    <>
      {editingId && (
        <button
          type="button"
          className={styles.buttonDanger}
          onClick={() => void handleDelete()}
          disabled={saving}
        >
          {t("spaceManager.delete")}
        </button>
      )}
      <div className={styles.footerActions}>
        <button type="button" className={styles.button} onClick={handleClose}>
          {t("spaceManager.cancel")}
        </button>
        {editorStep > firstStep && (
          <button
            type="button"
            className={styles.button}
            onClick={handleBack}
            disabled={saving}
          >
            {t("spaceManager.back")}
          </button>
        )}
        {editorStep < REVIEW_STEP ? (
          <button
            type="button"
            className={styles.buttonPrimary}
            onClick={handleNext}
            disabled={isNextDisabled}
          >
            {t("spaceManager.next")}
          </button>
        ) : (
          <button
            type="button"
            className={styles.buttonPrimary}
            onClick={() => void handleSave()}
            disabled={!canSave}
          >
            {t("spaceManager.save")}
          </button>
        )}
      </div>
    </>
  );
};

export default SpaceManagerFooter;
