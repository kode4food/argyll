import React from "react";
import Modal from "@/app/components/molecules/Modal";
import { useT } from "@/app/i18n";
import FeedManagerForm from "./FeedManagerForm";
import { FeedEditor, useFeedEditor } from "./useFeedEditor";
import styles from "@/app/styles/components/manager.module.css";

interface FeedManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

const titleKey = (isEditing: boolean, editingId: string | null): string => {
  if (!isEditing) return "feedManager.title";
  return editingId ? "feedManager.updateTitle" : "feedManager.createTitle";
};

interface FeedManagerFooterProps {
  editor: FeedEditor;
}

const FeedManagerFooter: React.FC<FeedManagerFooterProps> = ({ editor }) => {
  const t = useT();
  const { canSave, editingId, handleClose, isEditing, isSaving } = editor;

  if (!isEditing) {
    return (
      <>
        <button
          autoFocus
          type="button"
          className={styles.buttonPrimary}
          onClick={() => editor.openEditor()}
        >
          {t("feedManager.new")}
        </button>
        <button type="button" className={styles.button} onClick={handleClose}>
          {t("feedManager.close")}
        </button>
      </>
    );
  }

  return (
    <>
      {editingId && (
        <button
          type="button"
          className={styles.buttonDanger}
          onClick={() => void editor.handleDelete()}
          disabled={isSaving}
        >
          {t("feedManager.delete")}
        </button>
      )}
      <div className={styles.footerActions}>
        <button type="button" className={styles.button} onClick={handleClose}>
          {t("feedManager.cancel")}
        </button>
        <button
          type="button"
          className={styles.buttonPrimary}
          onClick={() => void editor.handleSave()}
          disabled={!canSave}
        >
          {t("feedManager.save")}
        </button>
      </div>
    </>
  );
};

const FeedManager: React.FC<FeedManagerProps> = ({ isOpen, onClose }) => {
  const t = useT();
  const editor = useFeedEditor(onClose);
  const { editingId, error, feeds, isEditing } = editor;

  return (
    <Modal
      isOpen={isOpen}
      onClose={editor.handleClose}
      title={t(titleKey(isEditing, editingId))}
      width={760}
      footer={<FeedManagerFooter editor={editor} />}
    >
      {isEditing ? (
        <div className={styles.editor}>
          <FeedManagerForm editor={editor} />
          {error && <div className={styles.error}>{error}</div>}
        </div>
      ) : (
        <div className={styles.list}>
          {feeds.length === 0 && (
            <div className={styles.empty}>{t("feedManager.empty")}</div>
          )}
          {feeds.map((feed) => (
            <button
              key={feed.id}
              type="button"
              className={styles.listItem}
              onClick={() => editor.openEditor(feed)}
            >
              <span>{feed.name}</span>
              <span className={styles.listItemId}>{feed.id}</span>
            </button>
          ))}
        </div>
      )}
    </Modal>
  );
};

export default FeedManager;
