import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { addTag, TAGIFY_SETTLE_DELAY_MS } from "@/app/testUtils/tags";
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

  test("enables editing and enforces suggestions when configured", async () => {
    const props = {
      Icon: IconAttributeLabel,
      label: "Tags",
      onChange,
      removeLabel: "Remove",
    };
    const { container, rerender } = render(
      <TagInput
        {...props}
        placeholder="No tools selected"
        readOnly
        suggestions={[]}
        tags={[]}
      />
    );
    const editor = container.querySelector(".tagify");
    const editorInput = editor?.querySelector(".tagify__input");
    expect(editor).toHaveAttribute("readonly");
    expect(editorInput).not.toHaveAttribute("contenteditable");
    expect(editorInput).toHaveAttribute(
      "data-placeholder",
      "No tools selected"
    );

    rerender(
      <TagInput
        {...props}
        placeholder="All tools"
        shouldEnforceSuggestions
        suggestions={suggestions}
        tags={["alpha"]}
      />
    );
    expect(editor).not.toHaveAttribute("readonly");
    expect(editorInput).toHaveAttribute("data-placeholder", "All tools");
    expect(tagTexts(container)).toEqual(["alpha"]);

    const input = screen.getByLabelText("All tools");
    await addTag(input, "zeta");
    await addTag(input, "beta");

    expect(tagTexts(container)).toEqual(["alpha", "beta"]);
  });

  test("renders the tags it was given", () => {
    const { container } = renderInput(["alpha", "beta"]);

    expect(tagTexts(container)).toEqual(["alpha", "beta"]);
  });

  test("labels each remove button with its own tag", () => {
    renderInput(["alpha"]);

    expect(screen.getByLabelText("Remove alpha")).toBeInTheDocument();
  });

  test("returns the caret to the input when a tag is clicked", () => {
    const { container } = renderInput(["alpha"]);

    fireEvent.click(container.querySelector("tag")!);

    expect(container.querySelector(".tagify__input")).toHaveFocus();
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
