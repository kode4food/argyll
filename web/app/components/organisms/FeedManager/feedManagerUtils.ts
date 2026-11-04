import {
  AttributeRole,
  AttributeType,
  Feed,
  SCRIPT_LANGUAGE_ARGYLL,
  ScriptConfig,
  Step,
} from "@/app/api";
import { validateDefaultValue } from "@/utils/stepUtils";

export interface FeedDraft {
  id: string;
  name: string;
  description: string;
  value: string;
  fold: ScriptConfig;
  attributes: string[];
  spaceId: string;
}

const DEFAULT_SEED = "0";

// Names the current value inside a fold, so no Attribute may use it
const FEED_CURRENT = "feed";

export const emptyDraft = (): FeedDraft => ({
  id: "",
  name: "",
  description: "",
  value: DEFAULT_SEED,
  fold: { language: SCRIPT_LANGUAGE_ARGYLL, script: "" },
  attributes: [],
  spaceId: "",
});

export const toDraft = (feed: Feed): FeedDraft => ({
  id: feed.id,
  name: feed.name,
  description: feed.description ?? "",
  value: feed.value,
  fold: { ...feed.fold },
  attributes: [...feed.attributes],
  spaceId: feed.space_id ?? "",
});

export const toFeed = (draft: FeedDraft): Feed => {
  const description = draft.description.trim();
  return {
    id: draft.id.trim(),
    name: draft.name.trim(),
    ...(description && { description }),
    value: draft.value.trim(),
    fold: { language: draft.fold.language, script: draft.fold.script.trim() },
    attributes: draft.attributes,
    ...(draft.spaceId && { space_id: draft.spaceId }),
  };
};

export const isDraftValid = (draft: FeedDraft): boolean => {
  const hasDetails = !!draft.id.trim() && !!draft.name.trim();
  const hasSeed =
    !!draft.value.trim() &&
    validateDefaultValue(draft.value, AttributeType.Any).valid;
  const hasAttributes =
    draft.attributes.length > 0 && !draft.attributes.includes(FEED_CURRENT);
  return hasDetails && hasSeed && hasAttributes && !!draft.fold.script.trim();
};

// Every Attribute some registered Step outputs, which is what a Feed can fold
export const outputNames = (steps: Step[]): string[] => {
  const names = new Set<string>();
  steps.forEach((step) =>
    Object.entries(step.attributes).forEach(([name, spec]) => {
      if (spec.role === AttributeRole.Output) names.add(name);
    })
  );
  return [...names].sort();
};

// The fold binds the current value and each arriving Attribute by name
export const foldCompletions = (draft: FeedDraft): string[] => [
  FEED_CURRENT,
  ...draft.attributes,
];
