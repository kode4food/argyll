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

const toDraft = (space: Space, timestamp: number): SpaceDraft => ({
  id: space.id,
  name: space.name,
  description: space.description ?? "",
  selector: Object.entries(space.selector?.match_labels ?? {}).map(
    ([key, value], index) => ({ id: `sel-${index}-${timestamp}`, key, value })
  ),
});

const emptyDraft = (): SpaceDraft => ({
  id: "",
  name: "",
  description: "",
  selector: [],
});

const toSpace = (draft: SpaceDraft): Space => {
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
};

const fingerprint = (space: Space) =>
  JSON.stringify({
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    selector: Object.entries(space.selector?.match_labels ?? {}).sort(),
  });

const SpaceManager: React.FC<SpaceManagerProps> = ({ isOpen, onClose }) => {
  const t = useT();
  const spaces = useSpaces();
  const { labelKeys, valuesForKey } = useLabelVocabulary();
  const spaceSelection = useSpaceSelection();

  const [draft, setDraft] = React.useState<SpaceDraft>(emptyDraft);
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);
  const selectorCounter = React.useRef(0);

  const selectSpace = (space: Space) => {
    setDraft(toDraft(space, Date.now()));
    setEditingId(space.id);
    setError(null);
  };

  const startNew = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setError(null);
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
        { id: `sel-${++selectorCounter.current}`, key: "", value: "" },
      ],
    }));
  };

  const removeSelector = (id: string) => {
    setDraft((current) => ({
      ...current,
      selector: current.selector.filter((pair) => pair.id !== id),
    }));
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
    const saved = await run(
      () =>
        editingId
          ? api.updateSpace(editingId, space)
          : api.registerSpace(space),
      "spaceManager.saveFailed"
    );
    if (saved) setEditingId(space.id);
  };

  const handleDelete = async () => {
    if (!editingId) return;
    const deleted = await run(
      () => api.unregisterSpace(editingId),
      "spaceManager.deleteFailed"
    );
    if (deleted) startNew();
  };

  const matchCount = editingId ? (spaceSelection[editingId]?.size ?? 0) : 0;
  const isValid =
    draft.id.trim().length > 0 &&
    draft.name.trim().length > 0 &&
    draft.selector.length > 0 &&
    draft.selector.every(({ key, value }) => key.trim() && value.trim());

  const editing = editingId
    ? spaces.find((space) => space.id === editingId)
    : undefined;
  const isDirty =
    !editing || fingerprint(toSpace(draft)) !== fingerprint(editing);

  const canSave = !saving && isValid && isDirty;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("spaceManager.title")}
      width={760}
      footer={
        <>
          <div className={styles.footerSelection}>
            <button type="button" className={styles.button} onClick={startNew}>
              {t("spaceManager.new")}
            </button>
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
          </div>
          <div className={styles.footerActions}>
            <button type="button" className={styles.button} onClick={onClose}>
              {t("spaceManager.close")}
            </button>
            <button
              type="button"
              className={styles.buttonPrimary}
              onClick={() => void handleSave()}
              disabled={!canSave}
            >
              {t("spaceManager.save")}
            </button>
          </div>
        </>
      }
    >
      <div className={styles.layout}>
        <div className={styles.list}>
          {spaces.length === 0 && (
            <div className={styles.empty}>{t("spaceManager.empty")}</div>
          )}
          {spaces.map((space) => (
            <button
              key={space.id}
              type="button"
              className={`${styles.listItem} ${
                editingId === space.id ? styles.listItemActive : ""
              }`}
              onClick={() => selectSpace(space)}
            >
              <span>{space.name}</span>
              <span className={styles.listItemId}>{space.id}</span>
            </button>
          ))}
        </div>

        <div className={styles.detail}>
          <div className={styles.field}>
            <label className={styles.label}>{t("spaceManager.idLabel")}</label>
            <input
              type="text"
              className={styles.input}
              value={draft.id}
              disabled={editingId !== null}
              onChange={(e) =>
                setDraft((current) => ({ ...current, id: e.target.value }))
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
                setDraft((current) => ({ ...current, name: e.target.value }))
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
              placeholder={t("spaceManager.descriptionPlaceholder")}
            />
          </div>

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

          {error && <div className={styles.error}>{error}</div>}
        </div>
      </div>
    </Modal>
  );
};

export default SpaceManager;
