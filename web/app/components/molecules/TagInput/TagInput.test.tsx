import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
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

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("opens the list from ArrowDown and highlights the first item", () => {
    renderInput();
    const input = screen.getByLabelText("add tag");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();

    fireEvent.keyDown(input, { key: "ArrowDown" });

    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "alpha" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("closes the list on Escape", () => {
    renderInput();
    const input = screen.getByLabelText("add tag");
    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    fireEvent.keyDown(input, { key: "Escape" });

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("wraps the highlight past both ends of the list", () => {
    renderInput();
    const input = screen.getByLabelText("add tag");
    fireEvent.keyDown(input, { key: "ArrowDown" });

    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(screen.getByRole("option", { name: "gamma" })).toHaveAttribute(
      "aria-selected",
      "true"
    );

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(screen.getByRole("option", { name: "alpha" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("commits the highlighted suggestion on Enter", () => {
    renderInput();
    const input = screen.getByLabelText("add tag");
    fireEvent.keyDown(input, { key: "ArrowDown" });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onChange).toHaveBeenCalledWith(["alpha"]);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("hover moves the highlight and mousedown commits", () => {
    renderInput();
    const input = screen.getByLabelText("add tag");
    fireEvent.keyDown(input, { key: "ArrowDown" });

    const beta = screen.getByRole("option", { name: "beta" });
    fireEvent.mouseEnter(beta);
    expect(beta).toHaveAttribute("aria-selected", "true");

    fireEvent.mouseDown(beta);
    expect(onChange).toHaveBeenCalledWith(["beta"]);
  });

  test("commits typed text on comma", () => {
    renderInput(["alpha"]);
    const input = screen.getByLabelText("add tag");
    fireEvent.change(input, { target: { value: " custom " } });
    fireEvent.keyDown(input, { key: "," });

    expect(onChange).toHaveBeenCalledWith(["alpha", "custom"]);
  });

  test("Backspace on an empty draft drops the last tag", () => {
    renderInput(["alpha", "beta"]);
    fireEvent.keyDown(screen.getByLabelText("add tag"), { key: "Backspace" });

    expect(onChange).toHaveBeenCalledWith(["alpha"]);
  });

  test("removes a tag from its remove button", () => {
    renderInput(["alpha", "beta"]);
    fireEvent.click(screen.getByRole("button", { name: "Remove alpha" }));

    expect(onChange).toHaveBeenCalledWith(["beta"]);
  });

  test("drops the placeholder once tags are present", () => {
    const { rerender } = renderInput();
    expect(screen.getByLabelText("add tag")).toHaveAttribute(
      "placeholder",
      "add tag"
    );

    rerender(
      <TagInput
        Icon={IconAttributeLabel}
        label="Tags"
        onChange={onChange}
        placeholder="add tag"
        removeLabel="Remove"
        suggestions={suggestions}
        tags={["alpha"]}
      />
    );

    expect(screen.getByLabelText("add tag")).toHaveAttribute("placeholder", "");
  });

  test("ignores a duplicate tag", () => {
    renderInput(["alpha"]);
    const input = screen.getByLabelText("add tag");
    fireEvent.change(input, { target: { value: "alpha" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onChange).not.toHaveBeenCalled();
  });
});
