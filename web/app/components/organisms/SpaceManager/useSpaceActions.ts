import React from "react";
import { api, Space } from "@/app/api";
import { useT } from "@/app/i18n";
import { useUI } from "@/app/contexts/UIContext";

export interface UseSpaceActionsOptions {
  editingId: string | null;
  onDeleted: () => void;
  onSaved: () => void;
  setError: React.Dispatch<React.SetStateAction<string | null>>;
  space: Space;
}

export interface SpaceActions {
  handleDelete: () => Promise<void>;
  handleSave: () => Promise<void>;
  saving: boolean;
}

export const useSpaceActions = ({
  editingId,
  onDeleted,
  onSaved,
  setError,
  space,
}: UseSpaceActionsOptions): SpaceActions => {
  const t = useT();
  const { setSpaceId } = useUI();
  const [saving, setSaving] = React.useState(false);

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
    if (saved) onSaved();
  };

  const handleDelete = async () => {
    if (!editingId) return;
    const deleted = await run(
      () => api.unregisterSpace(editingId),
      "spaceManager.deleteFailed"
    );
    if (deleted) onDeleted();
  };

  return { handleDelete, handleSave, saving };
};
