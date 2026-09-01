import React from "react";
import Modal from "@/app/components/molecules/Modal";
import { useT } from "@/app/i18n";
import { useTagVocabulary } from "@/app/hooks/useTagVocabulary";
import SpaceManagerDetailsStep from "./SpaceManagerDetailsStep";
import SpaceManagerFooter from "./SpaceManagerFooter";
import SpaceManagerReviewStep from "./SpaceManagerReviewStep";
import SpaceManagerSelectorStep from "./SpaceManagerSelectorStep";
import { EDITOR_STEPS, REVIEW_STEP } from "./spaceManagerUtils";
import { useSpaceEditor } from "./useSpaceEditor";
import styles from "./SpaceManager.module.css";

interface SpaceManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

const titleKey = (isEditing: boolean, editingId: string | null): string => {
  if (!isEditing) return "spaceManager.title";
  return editingId ? "spaceManager.updateTitle" : "spaceManager.createTitle";
};

const SpaceManager: React.FC<SpaceManagerProps> = ({ isOpen, onClose }) => {
  const t = useT();
  const tagVocabulary = useTagVocabulary();
  const editor = useSpaceEditor(onClose);
  const { currentPreview, draft, editingId, editorStep, error, isEditing } =
    editor;

  return (
    <Modal
      isOpen={isOpen}
      onClose={editor.handleClose}
      title={t(titleKey(isEditing, editingId))}
      width={760}
      footer={<SpaceManagerFooter editor={editor} />}
    >
      {isEditing ? (
        <div className={styles.editor}>
          <ol className={styles.stepper}>
            {EDITOR_STEPS.map((label, index) => (
              <li
                key={label}
                className={`${styles.step} ${
                  index === editorStep ? styles.stepActive : ""
                } ${index < editorStep ? styles.stepComplete : ""}`}
                aria-current={index === editorStep ? "step" : undefined}
              >
                {t(label)}
              </li>
            ))}
          </ol>

          {editorStep === 0 && (
            <SpaceManagerSelectorStep
              tags={draft.qbe}
              suggestions={tagVocabulary}
              onTagsChange={(qbe) => editor.patchDraft({ qbe })}
              onEditScript={editor.handleEditScript}
              isEditScriptDisabled={
                draft.qbe.length > 0 && (!editor.hasValidQBE || !currentPreview)
              }
            />
          )}

          {editorStep === REVIEW_STEP && (
            <SpaceManagerReviewStep
              selector={editor.reviewSelector}
              isScriptMode={draft.scriptMode}
              onScriptChange={(script) => editor.patchSelector({ script })}
              onLanguageChange={(language) =>
                editor.patchSelector({ language })
              }
              onEditScript={editor.handleEditScript}
            />
          )}

          {editorStep === 1 && (
            <SpaceManagerDetailsStep
              draft={draft}
              isIdLocked={editingId !== null}
              onPatch={editor.patchDraft}
              selectorDescription={editor.selectorDescription}
            />
          )}

          <div className={styles.matches}>
            {currentPreview &&
              t("spaceManager.matchingSteps", {
                count: currentPreview.step_ids.length,
              })}
          </div>

          {error && <div className={styles.error}>{error}</div>}
        </div>
      ) : (
        <div className={styles.list}>
          {editor.spaces.length === 0 && (
            <div className={styles.empty}>{t("spaceManager.empty")}</div>
          )}
          {editor.spaces.map((space) => (
            <button
              key={space.id}
              type="button"
              className={styles.listItem}
              onClick={() => editor.openEditor(space)}
            >
              <span>{space.name}</span>
              <span className={styles.listItemId}>{space.id}</span>
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
};

export default SpaceManager;
