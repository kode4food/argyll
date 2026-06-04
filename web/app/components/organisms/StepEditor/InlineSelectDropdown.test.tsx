import { fireEvent, render, screen } from "@testing-library/react";
import InlineSelectDropdown, {
  InlineSelectOption,
} from "./InlineSelectDropdown";

describe("InlineSelectDropdown", () => {
  const options: InlineSelectOption[] = [
    { value: "single", label: "Single" },
    { value: "multi", label: "Multi" },
    { value: "none", label: "None", disabled: true },
  ];
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("opens and marks the current value as selected", () => {
    render(
      <InlineSelectDropdown
        ariaLabel="collect mode"
        value="single"
        options={options}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "collect mode" }));

    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Single" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("lets mouse hover set the highlighted option", () => {
    render(
      <InlineSelectDropdown
        ariaLabel="collect mode"
        value="single"
        options={options}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "collect mode" }));

    const single = screen.getByRole("option", { name: "Single" });
    const multi = screen.getByRole("option", { name: "Multi" });
    expect(single.className).toContain("itemHighlighted");

    fireEvent.mouseEnter(multi);

    expect(multi.className).toContain("itemHighlighted");
    expect(single.className).not.toContain("itemHighlighted");
  });

  test("selects with Enter and skips disabled options with arrow keys", () => {
    render(
      <InlineSelectDropdown
        ariaLabel="collect mode"
        value="single"
        options={options}
        onChange={onChange}
      />
    );

    const button = screen.getByRole("button", { name: "collect mode" });
    button.focus();
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "Enter" });

    expect(onChange).toHaveBeenCalledWith("multi");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
