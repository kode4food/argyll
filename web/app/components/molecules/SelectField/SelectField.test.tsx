import { fireEvent, render, screen } from "@testing-library/react";
import { IconStandard } from "@/utils/iconRegistry";
import SelectField, { type SelectFieldOption } from "./SelectField";

describe("SelectField", () => {
  const options: SelectFieldOption[] = [
    { value: "first", label: "First", Icon: IconStandard },
    { value: "second", label: "Second", title: "The second" },
    { value: "third", label: "Third", disabled: true },
  ];
  const onChange = jest.fn();

  beforeEach(() => jest.clearAllMocks());

  const renderField = (value = "first") =>
    render(
      <SelectField
        ariaLabel="Choice"
        label="Choice"
        onChange={onChange}
        options={options}
        value={value}
      />
    );

  const face = () => screen.getByRole("button", { name: "Choice" });

  test("shows the selected option label", () => {
    renderField("second");

    expect(face()).toHaveTextContent("Second");
  });

  test("falls back to the raw value for an unknown option", () => {
    renderField("missing");

    expect(face()).toHaveTextContent("missing");
  });

  test("opens and closes the list", () => {
    renderField();
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();

    fireEvent.click(face());
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(face()).toHaveAttribute("aria-expanded", "true");

    fireEvent.click(face());
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("selects an option", () => {
    renderField();
    fireEvent.click(face());
    fireEvent.click(screen.getByRole("option", { name: "Second" }));

    expect(onChange).toHaveBeenCalledWith("second");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("marks the selected option and disables a disabled one", () => {
    renderField();
    fireEvent.click(face());

    expect(screen.getByRole("option", { name: "First" })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(screen.getByRole("option", { name: "Third" })).toBeDisabled();
  });

  test("selects with the keyboard", () => {
    renderField();
    fireEvent.click(face());
    fireEvent.keyDown(screen.getByRole("listbox").parentElement!, {
      key: "ArrowDown",
    });
    fireEvent.keyDown(screen.getByRole("listbox").parentElement!, {
      key: "Enter",
    });

    expect(onChange).toHaveBeenCalledWith("second");
  });

  test("renders without a label", () => {
    render(
      <SelectField
        ariaLabel="Choice"
        onChange={onChange}
        options={options}
        value="first"
      />
    );

    expect(screen.queryByText("Choice")).not.toBeInTheDocument();
    expect(face()).toBeInTheDocument();
  });
});
