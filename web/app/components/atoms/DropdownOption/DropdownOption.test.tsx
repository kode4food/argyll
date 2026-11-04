import { fireEvent, render, screen } from "@testing-library/react";
import DropdownOption from "./DropdownOption";

const renderOption = (props: Partial<Parameters<typeof DropdownOption>[0]>) => {
  const onHover = jest.fn();
  const onSelect = jest.fn();
  render(
    <DropdownOption
      isHighlighted={false}
      onHover={onHover}
      onSelect={onSelect}
      {...props}
    >
      Label
    </DropdownOption>
  );
  return {
    onHover,
    onSelect,
    option: screen.getByText("Label").closest("button")!,
  };
};

describe("DropdownOption", () => {
  test("selects and reports hover", () => {
    const { onHover, onSelect, option } = renderOption({ isActive: true });
    expect(option).toHaveAttribute("role", "option");
    expect(option).toHaveAttribute("aria-selected", "true");

    fireEvent.mouseEnter(option);
    fireEvent.click(option);
    expect(onHover).toHaveBeenCalled();
    expect(onSelect).toHaveBeenCalled();
  });

  test("ignores a disabled option", () => {
    const { onSelect, option } = renderOption({ isDisabled: true });
    fireEvent.click(option);
    expect(onSelect).not.toHaveBeenCalled();
  });

  test("leaves selection state off menu items", () => {
    const { option } = renderOption({ role: "menuitem" });
    expect(option).not.toHaveAttribute("aria-selected");
  });
});
