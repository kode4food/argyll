import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { TAGIFY_SETTLE_DELAY_MS } from "@/app/testUtils/tags";
import { IconAttributeLabel } from "@/utils/iconRegistry";
import TagInput from "./TagInput";

describe("TagInput", () => {
  const suggestions = ["alpha", "beta", "gamma"];
  const onChange = jest.fn();

  const renderInput = (tags: string[] = []) =>
    render(
      <TagInput
        Icon={IconAttributeLabel}
        label="Tags"
        onChange={onChange}
        placeholder="add tag"
        removeLabel="Remove"
        suggestions={suggestions}
        tags={tags}
      />
    );

  // Tagify blocks change events for a moment after loading its own value
  const settle = () =>
    act(() => new Promise((done) => setTimeout(done, TAGIFY_SETTLE_DELAY_MS)));

  const tagTexts = (container: HTMLElement) =>
    Array.from(container.querySelectorAll("tag")).map((tag) =>
      tag.textContent?.trim()
    );

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders the tags it was given", () => {
    const { container } = renderInput(["alpha", "beta"]);

    expect(tagTexts(container)).toEqual(["alpha", "beta"]);
  });

  test("labels each remove button with its own tag", () => {
    renderInput(["alpha"]);

    expect(screen.getByLabelText("Remove alpha")).toBeInTheDocument();
  });

  test("reports the remaining tags when one is removed", async () => {
    renderInput(["alpha", "beta"]);
    // Tagify holds back change events until its own load has settled
    await settle();

    fireEvent.click(screen.getByLabelText("Remove alpha"));

    await waitFor(() => expect(onChange).toHaveBeenCalledWith(["beta"]));
  });

  test("takes tags added from outside", () => {
    const { container, rerender } = renderInput(["alpha"]);

    rerender(
      <TagInput
        Icon={IconAttributeLabel}
        label="Tags"
        onChange={onChange}
        placeholder="add tag"
        removeLabel="Remove"
        suggestions={suggestions}
        tags={["alpha", "gamma"]}
      />
    );

    expect(tagTexts(container)).toEqual(["alpha", "gamma"]);
  });
});
