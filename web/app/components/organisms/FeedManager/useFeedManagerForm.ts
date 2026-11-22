import React from "react";
import { useT } from "@/app/i18n";
import { useSpaceOptions } from "@/app/hooks/useSpaceOptions";
import { useSteps, useWatchedFeeds } from "@/app/store/flowStore";
import { IconSpace } from "@/utils/iconRegistry";
import { foldCompletions, outputNames } from "./feedManagerUtils";
import { FeedEditor } from "./useFeedEditor";

export const useFeedManagerForm = (editor: FeedEditor) => {
  const t = useT();
  const watchedFeeds = useWatchedFeeds();
  const steps = useSteps();
  const { draft, editingId } = editor;
  const watchedFeed = editingId ? watchedFeeds[editingId] : undefined;
  const isIdLocked = editingId !== null;
  const outputs = React.useMemo(() => outputNames(steps), [steps]);
  const completions = React.useMemo(() => foldCompletions(draft), [draft]);
  const spaceOptions = useSpaceOptions(t("feedManager.spaceAny"), IconSpace);
  return { watchedFeed, isIdLocked, outputs, completions, spaceOptions };
};
