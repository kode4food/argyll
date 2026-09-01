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
  qbe: string[][];
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
    qbe: (space.qbe ?? []).map((term) => [...term]),
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
    qbe: [[]],
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
export function suggestedId(qbe: string[][]): string {
  return toQBE(qbe)
    .map((term) => term.join("-"))
    .join("-or-")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function suggestedName(qbe: string[][]): string {
  return toQBE(qbe)
    .map((term) => term.map(titleCase).join(" "))
    .join(" / ");
}

// Tags within a term read as a list, terms as alternatives
export function describeQBE(qbe: string[][]): string {
  return toQBE(qbe)
    .map((term) => term.join(", "))
    .join(" / ");
}

export function isDetailsValid(draft: SpaceDraft): boolean {
  return draft.id.trim().length > 0 && draft.name.trim().length > 0;
}

export function isQBEValid(draft: SpaceDraft): boolean {
  return (
    draft.qbe.length > 0 &&
    draft.qbe.every((term) => term.some((tag) => tag.trim()))
  );
}

// True once the draft carries a tag, whether or not every term has one yet
export function hasQBE(draft: SpaceDraft): boolean {
  return toQBE(draft.qbe).length > 0;
}

export function isSelectorValid(draft: SpaceDraft): boolean {
  return draft.scriptMode
    ? draft.selector.script.trim().length > 0
    : isQBEValid(draft);
}

export function isDraftValid(draft: SpaceDraft): boolean {
  return isDetailsValid(draft) && isSelectorValid(draft);
}

// Canonical form: every term sorted and deduped, then the terms themselves
// sorted and deduped, the same shape the engine stores, so the generated
// script and the dirty check both stay stable
export function toQBE(qbe: string[][]): string[][] {
  const terms = qbe
    .map((term) => [...new Set(term.map((tag) => tag.trim()).filter(Boolean))])
    .filter((term) => term.length > 0)
    .map((term) => term.sort())
    .sort(compareTerms);
  return terms.filter(
    (term, index) => index === 0 || compareTerms(terms[index - 1], term) !== 0
  );
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
    return qbeFingerprint(preview.space.qbe) === qbeFingerprint(draft.qbe);
  }
  return (
    preview.space.selector.language === draft.selector.language &&
    preview.space.selector.script === draft.selector.script.trim()
  );
}

function titleCase(tag: string): string {
  return tag
    .split(/[^a-zA-Z0-9]+/)
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(" ");
}

// Element-wise, with the shorter term first when one is a prefix of the
// other, matching how the engine orders them
function compareTerms(left: string[], right: string[]): number {
  for (let i = 0; i < left.length && i < right.length; i++) {
    if (left[i] !== right[i]) return left[i] < right[i] ? -1 : 1;
  }
  return left.length - right.length;
}

function qbeFingerprint(qbe?: string[][]): string {
  return JSON.stringify(toQBE(qbe ?? []));
}
