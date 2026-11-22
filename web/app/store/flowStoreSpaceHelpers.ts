import { Space } from "@/app/api";
import { compareByName } from "./flowStoreHelpers";

export interface CatalogSpaces {
  defined?: Record<string, Space>;
  selected?: Record<string, string[]>;
}

export const toCatalogSpaceState = (spaces: CatalogSpaces) => ({
  spaces: Object.values(spaces.defined ?? {}).sort(compareByName),
  spaceSelection: Object.fromEntries(
    Object.entries(spaces.selected ?? {}).map(([id, ids]) => [id, new Set(ids)])
  ),
});

export const assignStepSpaces = (
  selection: Record<string, Set<string>>,
  stepId: string,
  spaces: string[]
): Record<string, Set<string>> => {
  const next: Record<string, Set<string>> = {};
  Object.entries(selection).forEach(([spaceId, steps]) => {
    if (spaces.includes(spaceId) === steps.has(stepId)) {
      next[spaceId] = steps;
      return;
    }
    const updated = new Set(steps);
    if (spaces.includes(spaceId)) {
      updated.add(stepId);
    } else {
      updated.delete(stepId);
    }
    next[spaceId] = updated;
  });
  spaces.forEach((spaceId) => {
    if (!next[spaceId]) next[spaceId] = new Set([stepId]);
  });
  return next;
};
