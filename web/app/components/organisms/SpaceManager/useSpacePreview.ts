import React from "react";
import { api, SpacePreviewResponse } from "@/app/api";
import { useT } from "@/app/i18n";
import { useThrottledValue } from "@/app/contexts/useThrottledValue";
import { isSelectorValid, SpaceDraft, toSpace } from "./spaceManagerUtils";

const PREVIEW_THROTTLE_MS = 500;

export interface UseSpacePreviewOptions {
  draft: SpaceDraft;
  isEditing: boolean;
  setError: React.Dispatch<React.SetStateAction<string | null>>;
}

export interface SpacePreviewState {
  clearPreview: () => void;
  preview: SpacePreviewResponse | null;
}

// Previews the throttled draft, so typing in the selector does not put a
// request on every keystroke
export const useSpacePreview = ({
  draft,
  isEditing,
  setError,
}: UseSpacePreviewOptions): SpacePreviewState => {
  const t = useT();
  const [preview, setPreview] = React.useState<SpacePreviewResponse | null>(
    null
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
  }, [previewDraft, isEditing, setError, t]);

  return {
    clearPreview: () => setPreview(null),
    preview,
  };
};
