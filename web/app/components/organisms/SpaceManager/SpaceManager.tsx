import React from "react";
import {
  api,
  SCRIPT_LANGUAGE_LUA,
  ScriptConfig,
  Space,
  SpacePreviewResponse,
} from "@/app/api";
import Modal from "@/app/components/molecules/Modal";
import KeyValueTable, {
  type KeyValuePair,
} from "@/app/components/molecules/KeyValueTable";
import ScriptConfigEditor from "@/app/components/organisms/StepEditor/ScriptConfigEditor";
import { predicateLanguageOptions } from "@/app/components/organisms/StepEditor/stepEditorConstants";
import { useT } from "@/app/i18n";
import { IconAttributeLabel, IconPredicate } from "@/utils/iconRegistry";
import { useSpaces } from "@/app/store/flowStore";
import { useLabelVocabulary } from "@/app/hooks/useLabelVocabulary";
import { useThrottledValue } from "@/app/contexts/useThrottledValue";
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
  qbe: KeyValuePair[];
  selector: ScriptConfig;
  scriptMode: boolean;
}

const EDITOR_STEPS = [
  "spaceManager.selectorStep",
  "spaceManager.detailsStep",
  "spaceManager.reviewStep",
];
const REVIEW_STEP = EDITOR_STEPS.length - 1;
const PREVIEW_THROTTLE_MS = 500;

const SpaceManager: React.FC<SpaceManagerProps> = ({ isOpen, onClose }) => {
  const t = useT();
  const spaces = useSpaces();
  const { setSpaceId } = useUI();
  const { labelKeys, valuesForKey } = useLabelVocabulary();
  const [draft, setDraft] = React.useState<SpaceDraft>(emptyDraft);
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const [editorStep, setEditorStep] = React.useState(0);
  const [isEditing, setEditing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);
  const [preview, setPreview] = React.useState<SpacePreviewResponse | null>(
    null
  );
  const selectorCounterRef = React.useRef(0);
  const editing = editingId
    ? spaces.find((space) => space.id === editingId)
    : undefined;
  const space = toSpace(draft);
  const currentPreview = isCurrentPreview(preview, draft) ? preview : null;
  const reviewSelector = draft.scriptMode
    ? draft.selector
    : currentPreview?.space.selector;
  const isDirty = !editing || fingerprint(space) !== fingerprint(editing);

  const openEditor = (existing?: Space) => {
    const id = existing?.id ?? null;
    const nextDraft = existing ? toDraft(existing) : emptyDraft();
    setDraft(nextDraft);
    setEditingId(id);
    setEditorStep(nextDraft.scriptMode ? REVIEW_STEP : 0);
    setPreview(null);
    setError(null);
    setEditing(true);
  };

  const clearEditor = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setEditorStep(0);
    setPreview(null);
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
      qbe: current.qbe.map((pair) =>
        pair.id === id ? { ...pair, [field]: value } : pair
      ),
    }));
  };

  const addSelector = () => {
    setDraft((current) => ({
      ...current,
      qbe: [
        ...current.qbe,
        { id: `sel-${++selectorCounterRef.current}`, key: "", value: "" },
      ],
    }));
  };

  const removeSelector = (id: string) => {
    setDraft((current) => ({
      ...current,
      qbe: current.qbe.filter((pair) => pair.id !== id),
    }));
  };

  const editScript = () => {
    const selector = currentPreview?.space.selector;
    if (draft.qbe.length > 0 && !selector) return;
    setDraft((current) => ({
      ...current,
      qbe: [],
      selector: selector ?? current.selector,
      scriptMode: true,
    }));
    setEditorStep(REVIEW_STEP);
  };

  const handleNext = () => {
    if (editorStep === 0 && editingId === null) {
      setDraft((current) => {
        const suggested = selectorSuggestions(current.qbe);
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

  const hasValidDetails = isDetailsValid(draft);
  const hasValidQBE = isQBEValid(draft);
  const canSave = !saving && isDraftValid(draft) && isDirty;
  // Script mode skips the QBE builder, so Details is its first step
  const firstStep = draft.scriptMode ? 1 : 0;
  const selectorDescription = selectorSummary(draft.qbe);
  const selectorValueSuggestions = (key: string, id: string) =>
    valuesForKey(key).filter(
      (value) =>
        !draft.qbe.some(
          (pair) => pair.id !== id && pair.key === key && pair.value === value
        )
    );

  const previewDraft = useThrottledValue(draft, PREVIEW_THROTTLE_MS);

  React.useEffect(() => {
    if (!isEditing || !isSelectorValid(previewDraft)) {
      setPreview(null);
      return;
    }
    let active = true;
    setError(null);
    void api
      .previewSpace(toSpace(previewDraft))
      .then((result) => active && setPreview(result))
      .catch((err) => {
        if (active) {
          setPreview(null);
          setError(err?.message || t("spaceManager.previewFailed"));
        }
      });
    return () => {
      active = false;
    };
  }, [previewDraft, isEditing, t]);

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
        {editorStep > firstStep && (
          <button
            type="button"
            className={styles.button}
            onClick={() => setEditorStep((current) => current - 1)}
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
            disabled={
              saving ||
              (editorStep === 0
                ? !hasValidQBE
                : !hasValidDetails || (!draft.scriptMode && !currentPreview))
            }
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
      title={t(
        isEditing
          ? editingId
            ? "spaceManager.updateTitle"
            : "spaceManager.createTitle"
          : "spaceManager.title"
      )}
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
                canRepeatKeys
                label={t("spaceManager.selectorLabel")}
                addLabel={t("spaceManager.addSelector")}
                removeLabel={t("spaceManager.removeSelector")}
                keyPlaceholder={t("stepEditor.labelKeyPlaceholder")}
                valuePlaceholder={t("stepEditor.labelValuePlaceholder")}
                pairs={draft.qbe}
                onAdd={addSelector}
                onChange={updateSelector}
                onRemove={removeSelector}
                keySuggestions={labelKeys}
                valueSuggestions={selectorValueSuggestions}
              />
              <button
                type="button"
                className={styles.advanced}
                onClick={editScript}
                disabled={
                  draft.qbe.length > 0 && (!hasValidQBE || !currentPreview)
                }
              >
                {t("spaceManager.editScript")}
              </button>
            </div>
          )}

          {editorStep === REVIEW_STEP && (
            <div className={styles.detail}>
              <ScriptConfigEditor
                Icon={IconPredicate}
                label={t("spaceManager.selectorLabel")}
                value={reviewSelector?.script ?? ""}
                onChange={(script) =>
                  setDraft((current) => ({
                    ...current,
                    selector: { ...current.selector, script },
                  }))
                }
                language={reviewSelector?.language ?? SCRIPT_LANGUAGE_LUA}
                onLanguageChange={(language) =>
                  setDraft((current) => ({
                    ...current,
                    selector: { ...current.selector, language },
                  }))
                }
                languageOptions={predicateLanguageOptions}
                readOnly={!draft.scriptMode}
              />
              {!draft.scriptMode && (
                <button
                  type="button"
                  className={styles.advanced}
                  onClick={editScript}
                >
                  {t("spaceManager.editScript")}
                </button>
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
    qbe: Object.entries(space.qbe ?? {}).flatMap(([key, values]) =>
      values.map((value, index) => ({
        id: `${key}-${index}`,
        key,
        value,
      }))
    ),
    selector: space.selector
      ? { ...space.selector }
      : { language: SCRIPT_LANGUAGE_LUA, script: "" },
    scriptMode: !space.qbe || Object.keys(space.qbe).length === 0,
  };
}

function emptyDraft(): SpaceDraft {
  return {
    id: "",
    name: "",
    description: "",
    qbe: [],
    selector: { language: SCRIPT_LANGUAGE_LUA, script: "" },
    scriptMode: false,
  };
}

function toSpace(draft: SpaceDraft): Space {
  const qbe = toQBE(draft.qbe);
  return {
    id: draft.id.trim(),
    name: draft.name.trim(),
    ...(draft.description.trim() && { description: draft.description.trim() }),
    ...(draft.scriptMode
      ? {
          selector: { ...draft.selector, script: draft.selector.script.trim() },
        }
      : { qbe }),
  };
}

function isDetailsValid(draft: SpaceDraft): boolean {
  return draft.id.trim().length > 0 && draft.name.trim().length > 0;
}

function isQBEValid(draft: SpaceDraft): boolean {
  return (
    draft.qbe.length > 0 &&
    draft.qbe.every(({ key, value }) => key.trim() && value.trim())
  );
}

function isSelectorValid(draft: SpaceDraft): boolean {
  return draft.scriptMode
    ? draft.selector.script.trim().length > 0
    : isQBEValid(draft);
}

function isDraftValid(draft: SpaceDraft): boolean {
  return isDetailsValid(draft) && isSelectorValid(draft);
}

// Canonical form: values sorted and deduped, the same shape the engine
// stores, so the generated script and the dirty check both stay stable
function toQBE(pairs: KeyValuePair[]): Record<string, string[]> {
  const qbe: Record<string, string[]> = {};
  pairs.forEach(({ key, value }) => {
    const name = key.trim();
    const match = value.trim();
    if (name && !qbe[name]?.includes(match)) {
      (qbe[name] ??= []).push(match);
    }
  });
  Object.values(qbe).forEach((values) => values.sort());
  return qbe;
}

function fingerprint(space: Space): string {
  return JSON.stringify({
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    selector: space.qbe ? undefined : space.selector,
    qbe: qbeFingerprint(space.qbe),
  });
}

function isCurrentPreview(
  preview: SpacePreviewResponse | null,
  draft: SpaceDraft
): boolean {
  if (!preview) return false;
  if (!draft.scriptMode) {
    return (
      qbeFingerprint(preview.space.qbe) === qbeFingerprint(toQBE(draft.qbe))
    );
  }
  return (
    preview.space.selector.language === draft.selector.language &&
    preview.space.selector.script === draft.selector.script.trim()
  );
}

function qbeFingerprint(qbe?: Record<string, string[]>): string {
  return JSON.stringify(Object.entries(qbe ?? {}).sort());
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
