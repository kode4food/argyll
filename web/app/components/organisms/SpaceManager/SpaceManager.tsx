import React from "react";
import { api, Space } from "@/app/api";
import Modal from "@/app/components/molecules/Modal";
import KeyValueTable, {
  type KeyValuePair,
} from "@/app/components/molecules/KeyValueTable";
import { useT } from "@/app/i18n";
import { IconAttributeLabel } from "@/utils/iconRegistry";
import { useSpaces, useSpaceSelection } from "@/app/store/flowStore";
import { useLabelVocabulary } from "@/app/hooks/useLabelVocabulary";
import { useUI } from "@/app/contexts/UIContext";
import styles from "./SpaceManager.module.css";

interface SpaceManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

interface SpaceDraft {
  id: string;
  name: string;
  description: string;
  selector: KeyValuePair[];
}

const EDITOR_STEPS = ["spaceManager.selectorStep", "spaceManager.detailsStep"];

const SpaceManager: React.FC<SpaceManagerProps> = ({ isOpen, onClose }) => {
  const t = useT();
  const spaces = useSpaces();
  const { setSpaceId } = useUI();
  const { labelKeys, valuesForKey } = useLabelVocabulary();
  const spaceSelection = useSpaceSelection();
  const [draft, setDraft] = React.useState<SpaceDraft>(emptyDraft);
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const [editorStep, setEditorStep] = React.useState(0);
  const [isEditing, setEditing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);
  const selectorCounterRef = React.useRef(0);
  const editing = editingId
    ? spaces.find((space) => space.id === editingId)
    : undefined;
  const isDirty =
    !editing || fingerprint(toSpace(draft)) !== fingerprint(editing);

  const openEditor = (space?: Space) => {
    const id = space?.id ?? null;
    setDraft(space ? toDraft(space) : emptyDraft());
    setEditingId(id);
    setEditorStep(0);
    setError(null);
    setEditing(true);
  };

  const clearEditor = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setEditorStep(0);
    setError(null);
    setEditing(false);
  };

  const handleClose = () => {
    clearEditor();
    onClose();
  };

  const updateSelector = (
    id: string,
    field: "key" | "value",
    value: string
  ) => {
    setDraft((current) => ({
      ...current,
      selector: current.selector.map((pair) =>
        pair.id === id ? { ...pair, [field]: value } : pair
      ),
    }));
  };

  const addSelector = () => {
    setDraft((current) => ({
      ...current,
      selector: [
        ...current.selector,
        { id: `sel-${++selectorCounterRef.current}`, key: "", value: "" },
      ],
    }));
  };

  const removeSelector = (id: string) => {
    setDraft((current) => ({
      ...current,
      selector: current.selector.filter((pair) => pair.id !== id),
    }));
  };

  const handleNext = () => {
    if (editingId === null) {
      setDraft((current) => {
        const suggested = selectorSuggestions(current.selector);
        return {
          ...current,
          id: current.id || suggested.id,
          name: current.name || suggested.name,
        };
      });
    }
    setEditorStep((current) => current + 1);
  };

  const run = async (action: () => Promise<unknown>, failure: string) => {
    setSaving(true);
    setError(null);
    try {
      await action();
      return true;
    } catch (err: any) {
      setError(err?.message || t(failure));
      return false;
    } finally {
      setSaving(false);
    }
  };

  const handleSave = async () => {
    const space = toSpace(draft);
    const saved = await run(async () => {
      const selected = editingId
        ? api.updateSpace(editingId, space)
        : api.registerSpace(space);
      setSpaceId((await selected).id);
    }, "spaceManager.saveFailed");
    if (saved) {
      clearEditor();
      onClose();
    }
  };

  const handleDelete = async () => {
    if (!editingId) return;
    const deleted = await run(
      () => api.unregisterSpace(editingId),
      "spaceManager.deleteFailed"
    );
    if (deleted) clearEditor();
  };

  const detailsValid =
    draft.id.trim().length > 0 && draft.name.trim().length > 0;
  const selectorValid =
    draft.selector.length > 0 &&
    draft.selector.every(({ key, value }) => key.trim() && value.trim());
  const isValid = detailsValid && selectorValid;
  const canSave = !saving && isValid && isDirty;
  const matchCount = editingId ? (spaceSelection[editingId]?.size ?? 0) : 0;
  const selectorDescription = selectorSummary(draft.selector);

  const footer = isEditing ? (
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
        {editorStep > 0 && (
          <button
            type="button"
            className={styles.button}
            onClick={() => setEditorStep((current) => current - 1)}
            disabled={saving}
          >
            {t("spaceManager.back")}
          </button>
        )}
        {editorStep < EDITOR_STEPS.length - 1 ? (
          <button
            type="button"
            className={styles.buttonPrimary}
            onClick={handleNext}
            disabled={saving || !selectorValid}
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
  ) : (
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

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={t("spaceManager.title")}
      width={760}
      footer={footer}
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
            <div className={styles.detail}>
              <KeyValueTable
                Icon={IconAttributeLabel}
                label={t("spaceManager.selectorLabel")}
                addLabel={t("spaceManager.addSelector")}
                removeLabel={t("spaceManager.removeSelector")}
                keyPlaceholder={t("stepEditor.labelKeyPlaceholder")}
                valuePlaceholder={t("stepEditor.labelValuePlaceholder")}
                pairs={draft.selector}
                onAdd={addSelector}
                onChange={updateSelector}
                onRemove={removeSelector}
                keySuggestions={labelKeys}
                valueSuggestions={valuesForKey}
              />
              {editingId && (
                <div className={styles.matches}>
                  {t("overview.stepsRegistered", { count: matchCount })}
                </div>
              )}
            </div>
          )}

          {editorStep === 1 && (
            <div className={styles.detail}>
              <div className={styles.field}>
                <label className={styles.label}>
                  {t("spaceManager.idLabel")}
                </label>
                <input
                  type="text"
                  className={styles.input}
                  value={draft.id}
                  disabled={editingId !== null}
                  onChange={(e) =>
                    setDraft((current) => ({
                      ...current,
                      id: e.target.value,
                    }))
                  }
                  placeholder={t("spaceManager.idPlaceholder")}
                />
              </div>

              <div className={styles.field}>
                <label className={styles.label}>
                  {t("spaceManager.nameLabel")}
                </label>
                <input
                  type="text"
                  className={styles.input}
                  value={draft.name}
                  onChange={(e) =>
                    setDraft((current) => ({
                      ...current,
                      name: e.target.value,
                    }))
                  }
                  placeholder={t("spaceManager.namePlaceholder")}
                />
              </div>

              <div className={styles.field}>
                <label className={styles.label}>
                  {t("spaceManager.descriptionLabel")}
                </label>
                <input
                  type="text"
                  className={styles.input}
                  value={draft.description}
                  onChange={(e) =>
                    setDraft((current) => ({
                      ...current,
                      description: e.target.value,
                    }))
                  }
                  placeholder={
                    selectorDescription
                      ? t("spaceManager.descriptionSuggestion", {
                          selector: selectorDescription,
                        })
                      : t("spaceManager.descriptionPlaceholder")
                  }
                />
              </div>
            </div>
          )}

          {error && <div className={styles.error}>{error}</div>}
        </div>
      ) : (
        <div className={styles.list}>
          {spaces.length === 0 && (
            <div className={styles.empty}>{t("spaceManager.empty")}</div>
          )}
          {spaces.map((space) => (
            <button
              key={space.id}
              type="button"
              className={styles.listItem}
              onClick={() => openEditor(space)}
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

function toDraft(space: Space): SpaceDraft {
  return {
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    selector: Object.entries(space.selector?.match_labels ?? {}).map(
      ([key, value]) => ({ id: key, key, value })
    ),
  };
}

function emptyDraft(): SpaceDraft {
  return { id: "", name: "", description: "", selector: [] };
}

function toSpace(draft: SpaceDraft): Space {
  const matchLabels: Record<string, string> = {};
  draft.selector.forEach(({ key, value }) => {
    const name = key.trim();
    if (name) matchLabels[name] = value.trim();
  });
  return {
    id: draft.id.trim(),
    name: draft.name.trim(),
    ...(draft.description.trim() && { description: draft.description.trim() }),
    selector: { match_labels: matchLabels },
  };
}

function fingerprint(space: Space): string {
  return JSON.stringify({
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    selector: Object.entries(space.selector?.match_labels ?? {}).sort(),
  });
}

function selectorSuggestions(pairs: KeyValuePair[]) {
  return {
    id: pairs
      .flatMap(({ key, value }) => [key.trim(), value.trim()])
      .join("-")
      .toLowerCase()
      .replace(/[^a-z0-9_.+ -]/g, "")
      .replaceAll(" ", "-")
      .replace(/^-+|-+$/g, ""),
    name: pairs
      .map(
        ({ key, value }) => `${humanize(value.trim())} ${humanize(key.trim())}`
      )
      .join(" / "),
  };
}

function humanize(value: string): string {
  return value.replace(/[-_]+/g, " ");
}

function selectorSummary(pairs: KeyValuePair[]): string {
  return pairs
    .map(({ key, value }) => `${key.trim()}=${value.trim()}`)
    .join(", ");
}

export default SpaceManager;
