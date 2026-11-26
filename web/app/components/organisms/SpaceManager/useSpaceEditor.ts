import React from "react";
import { ScriptConfig, Space } from "@/app/api";
import { useSpaces } from "@/app/store/flowStore";
import {
  describeQBE,
  emptyDraft,
  hasQBE,
  isCurrentPreview,
  isDetailsValid,
  isDraftValid,
  isQBEValid,
  REVIEW_STEP,
  SpaceDraft,
  suggestedId,
  suggestedName,
  toDraft,
  toSpace,
} from "./spaceManagerUtils";
import { useSpaceActions } from "./useSpaceActions";
import { useSpacePreview } from "./useSpacePreview";

export type SpaceEditor = ReturnType<typeof useSpaceEditor>;

export const useSpaceEditor = (onClose: () => void, spaceId?: string) => {
  const spaces = useSpaces();
  const existing = spaces.find((space) => space.id === spaceId);
  const [draft, setDraft] = React.useState<SpaceDraft>(() =>
    existing ? toDraft(existing) : emptyDraft()
  );
  const [editingId, setEditingId] = React.useState(existing?.id ?? null);
  const [editorStep, setEditorStep] = React.useState(
    existing && draft.scriptMode ? REVIEW_STEP : 0
  );
  const [isEditing, setEditing] = React.useState(existing !== undefined);
  const [initialData, setInitialData] = React.useState(() =>
    JSON.stringify(toSpace(draft))
  );
  const [error, setError] = React.useState<string | null>(null);

  const { clearPreview, preview } = useSpacePreview({
    draft,
    isEditing,
    setError,
  });

  const space = toSpace(draft);
  const currentPreview = isCurrentPreview(preview, draft) ? preview : null;
  const isDirty = JSON.stringify(space) !== initialData;

  const clearEditor = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setEditorStep(0);
    clearPreview();
    setError(null);
    setEditing(false);
  };

  const handleClose = () => {
    clearEditor();
    onClose();
  };

  const { handleDelete, handleSave, saving } = useSpaceActions({
    editingId,
    onDeleted: clearEditor,
    onSaved: handleClose,
    setError,
    space,
  });

  const openEditor = (existing?: Space) => {
    const nextDraft = existing ? toDraft(existing) : emptyDraft();
    setDraft(nextDraft);
    setInitialData(JSON.stringify(toSpace(nextDraft)));
    setEditingId(existing?.id ?? null);
    setEditorStep(nextDraft.scriptMode ? REVIEW_STEP : 0);
    clearPreview();
    setError(null);
    setEditing(true);
  };

  const patchDraft = (patch: Partial<SpaceDraft>) => {
    setDraft((current) => ({ ...current, ...patch }));
  };

  const patchSelector = (patch: Partial<ScriptConfig>) => {
    setDraft((current) => ({
      ...current,
      selector: { ...current.selector, ...patch },
    }));
  };

  const handleEditScript = () => {
    const selector = currentPreview?.space.selector;
    if (hasQBE(draft) && !selector) return;
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
      setDraft((current) => ({
        ...current,
        id: current.id || suggestedId(current.qbe),
        name: current.name || suggestedName(current.qbe),
      }));
    }
    setEditorStep((current) => current + 1);
  };

  return {
    canSave: !saving && isDraftValid(draft) && isDirty,
    currentPreview,
    draft,
    editingId,
    editorStep,
    error,
    // Script mode skips the QBE builder, so Details is its first step
    firstStep: draft.scriptMode ? 1 : 0,
    handleBack: () => setEditorStep((current) => current - 1),
    handleClose,
    handleDelete,
    handleEditScript,
    handleNext,
    handleSave,
    hasQBE: hasQBE(draft),
    hasValidDetails: isDetailsValid(draft),
    hasValidQBE: isQBEValid(draft),
    isDirty,
    isEditing,
    openEditor,
    patchDraft,
    patchSelector,
    reviewSelector: draft.scriptMode
      ? draft.selector
      : currentPreview?.space.selector,
    saving,
    selectorDescription: describeQBE(draft.qbe),
    spaces,
  };
};
