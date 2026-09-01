import {
  SCRIPT_LANGUAGE_LUA,
  ScriptConfig,
  Space,
  SpacePreviewResponse,
} from "@/app/api";

export interface SpaceDraft {
  id: string;
  name: string;
  description: string;
  qbe: string[];
  selector: ScriptConfig;
  scriptMode: boolean;
}

export const EDITOR_STEPS = [
  "spaceManager.selectorStep",
  "spaceManager.detailsStep",
  "spaceManager.reviewStep",
];
export const REVIEW_STEP = EDITOR_STEPS.length - 1;

export function toDraft(space: Space): SpaceDraft {
  return {
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    qbe: [...(space.qbe ?? [])],
    selector: space.selector
      ? { ...space.selector }
      : { language: SCRIPT_LANGUAGE_LUA, script: "" },
    scriptMode: !space.qbe || space.qbe.length === 0,
  };
}

export function emptyDraft(): SpaceDraft {
  return {
    id: "",
    name: "",
    description: "",
    qbe: [],
    selector: { language: SCRIPT_LANGUAGE_LUA, script: "" },
    scriptMode: false,
  };
}

export function toSpace(draft: SpaceDraft): Space {
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

// Tags are opaque, so the suggestions treat every run of non-alphanumerics as
// a separator rather than reading any structure into them
export function suggestedId(tags: string[]): string {
  return toQBE(tags)
    .join("-")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function suggestedName(tags: string[]): string {
  return toQBE(tags)
    .map((tag) =>
      tag
        .split(/[^a-zA-Z0-9]+/)
        .filter(Boolean)
        .map((word) => word[0].toUpperCase() + word.slice(1))
        .join(" ")
    )
    .join(" / ");
}

export function isDetailsValid(draft: SpaceDraft): boolean {
  return draft.id.trim().length > 0 && draft.name.trim().length > 0;
}

export function isQBEValid(draft: SpaceDraft): boolean {
  return draft.qbe.length > 0 && draft.qbe.every((tag) => tag.trim());
}

export function isSelectorValid(draft: SpaceDraft): boolean {
  return draft.scriptMode
    ? draft.selector.script.trim().length > 0
    : isQBEValid(draft);
}

export function isDraftValid(draft: SpaceDraft): boolean {
  return isDetailsValid(draft) && isSelectorValid(draft);
}

// Canonical form: sorted and deduped, the same shape the engine stores, so
// the generated script and the dirty check both stay stable
export function toQBE(tags: string[]): string[] {
  return [...new Set(tags.map((tag) => tag.trim()).filter(Boolean))].sort();
}

export function fingerprint(space: Space): string {
  return JSON.stringify({
    id: space.id,
    name: space.name,
    description: space.description ?? "",
    selector: space.qbe ? undefined : space.selector,
    qbe: qbeFingerprint(space.qbe),
  });
}

export function isCurrentPreview(
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

function qbeFingerprint(qbe?: string[]): string {
  return JSON.stringify([...(qbe ?? [])].sort());
}
