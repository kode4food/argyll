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

describe("InlineSelectDropdown with an icon face", () => {
  const options: InlineSelectOption[] = [
    { value: "a", label: "Alpha", icon: <span>A</span> },
    { value: "b", label: "Beta", icon: <span>B</span> },
    { value: "c", label: "Gamma", icon: <span>C</span> },
  ];
  const onChange = jest.fn();
  const faceIcon = <span>icon</span>;

  const renderIconFace = (value: string) =>
    render(
      <InlineSelectDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value={value}
        options={options}
        onChange={onChange}
      />
    );

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders the icon as its face, titled with the choice", () => {
    renderIconFace("a");
    const face = screen.getByRole("button", { name: "choose role" });
    expect(face).toHaveTextContent("icon");
    expect(face).toHaveAttribute("title", "Alpha");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("lists each option with its icon", () => {
    renderIconFace("b");
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    expect(screen.getByRole("listbox", { name: "choose role" })).toBeVisible();
    expect(screen.getAllByRole("option")).toHaveLength(3);
    expect(screen.getByRole("option", { name: /Beta/ })).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("calls onChange and closes on option click", () => {
    renderIconFace("a");
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    fireEvent.click(screen.getByRole("option", { name: /Gamma/ }));
    expect(onChange).toHaveBeenCalledWith("c");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("closes on outside mousedown", () => {
    render(
      <div>
        <InlineSelectDropdown
          ariaLabel="choose role"
          faceIcon={faceIcon}
          value="a"
          options={options}
          onChange={onChange}
        />
        <button data-testid="outside">outside</button>
      </div>
    );
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("toggle closes open list", () => {
    renderIconFace("a");
    const face = screen.getByRole("button", { name: "choose role" });
    fireEvent.click(face);
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.click(face);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
