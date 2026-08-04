import { fireEvent, render, screen } from "@testing-library/react";
import { IconCompensate } from "@/utils/iconRegistry";
import IconCheckbox from "./IconCheckbox";

describe("IconCheckbox", () => {
  const baseProps = {
    checked: false,
    Icon: IconCompensate,
    label: "Compensate",
    onChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders the label and unchecked state", () => {
    render(<IconCheckbox {...baseProps} />);

    expect(screen.getByText("Compensate")).toBeInTheDocument();
    expect(screen.getByRole("checkbox")).not.toBeChecked();
  });

  test("reports the new value on toggle", () => {
    render(<IconCheckbox {...baseProps} checked={true} />);

    fireEvent.click(screen.getByRole("checkbox"));

    expect(baseProps.onChange).toHaveBeenCalledWith(false);
  });
});
