import React from "react";
import { api, Feed, ScriptConfig } from "@/app/api";
import { useCatalogAction } from "@/app/hooks/useCatalogAction";
import { useT } from "@/app/i18n";
import { useFeeds, useFlowStore } from "@/app/store/flowStore";
import {
  emptyDraft,
  FeedDraft,
  isDraftValid,
  toDraft,
  toFeed,
} from "./feedManagerUtils";

export type FeedEditor = ReturnType<typeof useFeedEditor>;

export const useFeedEditor = (onClose: () => void) => {
  const t = useT();
  const feeds = useFeeds();
  const setWatchedFeeds = useFlowStore((state) => state.setWatchedFeeds);
  const [draft, setDraft] = React.useState<FeedDraft>(emptyDraft);
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const [isEditing, setEditing] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const { isSaving, run } = useCatalogAction(setError);

  // Watches every Feed while the manager is open, reloading when a definition
  // change starts a new version
  React.useEffect(() => {
    let active = true;
    Promise.all(feeds.map((feed) => api.getFeed(feed.id)))
      .then((responses) => {
        if (!active) return;
        setWatchedFeeds(
          Object.fromEntries(responses.map(({ state }) => [state.id, state]))
        );
      })
      .catch((err: any) => {
        if (active) setError(err?.message || t("feedManager.loadFailed"));
      });
    return () => {
      active = false;
      setWatchedFeeds({});
    };
  }, [feeds, setWatchedFeeds, t]);

  const clearEditor = () => {
    setDraft(emptyDraft());
    setEditingId(null);
    setError(null);
    setEditing(false);
  };

  const handleClose = () => {
    clearEditor();
    onClose();
  };

  const openEditor = (existing?: Feed) => {
    setDraft(existing ? toDraft(existing) : emptyDraft());
    setEditingId(existing?.id ?? null);
    setError(null);
    setEditing(true);
  };

  const handleSave = async () => {
    const feed = toFeed(draft);
    const saved = await run(
      () =>
        editingId ? api.updateFeed(editingId, feed) : api.registerFeed(feed),
      "feedManager.saveFailed"
    );
    if (saved) clearEditor();
  };

  const handleDelete = async () => {
    if (!editingId) return;
    const deleted = await run(
      () => api.unregisterFeed(editingId),
      "feedManager.deleteFailed"
    );
    if (deleted) clearEditor();
  };

  const patchDraft = (patch: Partial<FeedDraft>) => {
    setDraft((current) => ({ ...current, ...patch }));
  };

  const patchFold = (patch: Partial<ScriptConfig>) => {
    setDraft((current) => ({
      ...current,
      fold: { ...current.fold, ...patch },
    }));
  };

  return {
    canSave: !isSaving && isDraftValid(draft),
    draft,
    editingId,
    error,
    feeds,
    handleClose,
    handleDelete,
    handleSave,
    isEditing,
    openEditor,
    patchDraft,
    patchFold,
    isSaving,
  };
};
