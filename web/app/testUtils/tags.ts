import { act } from "@testing-library/react";

interface TagifyInstance {
  addTags: (tags: string[]) => unknown;
  whitelist: string[];
}

export const TAGIFY_SETTLE_DELAY_MS = 150;

const instanceOf = (input: HTMLElement) =>
  (input as HTMLElement & { __tagify?: TagifyInstance }).__tagify;

export const suggestionsOf = (input: HTMLElement) =>
  instanceOf(input)?.whitelist;

// Tagify types into a contenteditable that jsdom cannot drive, so tests reach
// the field through the instance Tagify hangs off the original input. It also
// holds back change events briefly after loading, hence the wait
export const addTag = async (input: HTMLElement, value: string) => {
  await act(async () => {
    instanceOf(input)?.addTags([value]);
    await new Promise((done) => setTimeout(done, TAGIFY_SETTLE_DELAY_MS));
  });
};
