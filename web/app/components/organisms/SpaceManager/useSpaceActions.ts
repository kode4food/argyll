import React from "react";
import { api, Space } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import { useCatalogAction } from "@/app/hooks/useCatalogAction";

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
  const { setSpaceId } = useUI();
  const { isSaving: saving, run } = useCatalogAction(setError);

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
