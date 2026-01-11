import { render, screen, fireEvent } from "@testing-library/react";
import DurationInput from "./DurationInput";

// Mock the ms library to make tests more predictable
jest.mock("ms", () => {
  const originalMs = jest.requireActual("ms");
  return {
    __esModule: true,
    default: (value: string | number, options?: { long?: boolean }) => {
      if (typeof value === "number") {
        // Formatting milliseconds to string
        if (options?.long) {
          if (value === 5000) return "5 seconds";
          if (value === 86400000) return "1 day";
          if (value === 93784000) return "1 day";
          if (value === 172805000) return "2 days";
          if (value === 10800000) return "3 hours";
          if (value === 1800000) return "30 minutes";
          if (value === 45000) return "45 seconds";
          if (value === 60000) return "1 minute";
          if (value === 2592000000) return "30 days";
        }
        return originalMs(value, options);
      }
      // Parsing string to milliseconds
      return originalMs(value);
    },
  };
});

describe("DurationInput", () => {
  test("renders with initial millisecond value converted to readable string", () => {
    render(<DurationInput value={5000} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    expect(input).toHaveValue("5 seconds");
  });

  test("converts complex duration correctly", () => {
    const ms = (24 * 60 + 2 * 60 + 3) * 60 * 1000 + 4 * 1000; // 93784000
    render(<DurationInput value={ms} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    expect(input).toHaveValue("1 day");
  });

  test("calls onChange with correct milliseconds when user types '2 days'", () => {
    const onChange = jest.fn();
    render(<DurationInput value={5000} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "2 days" } });

    expect(onChange).toHaveBeenCalledWith(172800000); // 2 * 24 * 60 * 60 * 1000
  });

  test("calls onChange with correct milliseconds when user types '3h'", () => {
    const onChange = jest.fn();
    render(<DurationInput value={0} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "3h" } });

    expect(onChange).toHaveBeenCalledWith(10800000); // 3 * 60 * 60 * 1000
  });

  test("calls onChange with correct milliseconds when user types '30m'", () => {
    const onChange = jest.fn();
    render(<DurationInput value={0} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "30m" } });

    expect(onChange).toHaveBeenCalledWith(1800000); // 30 * 60 * 1000
  });

  test("calls onChange with correct milliseconds when user types '45s'", () => {
    const onChange = jest.fn();
    render(<DurationInput value={0} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "45s" } });

    expect(onChange).toHaveBeenCalledWith(45000); // 45 * 1000
  });

  test("displays placeholder text", () => {
    render(<DurationInput value={0} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    expect(input).toHaveAttribute(
      "placeholder",
      "e.g. 5d, 2 days 3h, 1.5 days"
    );
  });

  test("handles empty input values as zero", () => {
    const onChange = jest.fn();
    render(<DurationInput value={5000} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "" } });

    expect(onChange).toHaveBeenCalledWith(0);
  });

  test("updates when value prop changes", () => {
    const { rerender } = render(
      <DurationInput value={5000} onChange={jest.fn()} />
    );

    let input = screen.getByRole("textbox");
    expect(input).toHaveValue("5 seconds");

    rerender(<DurationInput value={60000} onChange={jest.fn()} />);

    input = screen.getByRole("textbox");
    expect(input).toHaveValue("1 minute");
  });

  test("supports long durations (30 days)", () => {
    const thirtyDays = 30 * 24 * 60 * 60 * 1000;
    render(<DurationInput value={thirtyDays} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    expect(input).toHaveValue("30 days");
  });

  test("handles decimal durations '1.5 days'", () => {
    const onChange = jest.fn();
    render(<DurationInput value={0} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "1.5 days" } });

    expect(onChange).toHaveBeenCalledWith(129600000); // 1.5 * 24 * 60 * 60 * 1000
  });

  test("shows invalid state for unparseable input", () => {
    render(<DurationInput value={5000} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "not a valid duration" } });

    expect(input).toHaveClass("invalid");
  });

  test("shows invalid state for negative duration", () => {
    render(<DurationInput value={5000} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "-5 seconds" } });

    expect(input).toHaveClass("invalid");
  });

  test("handles focus event", () => {
    render(<DurationInput value={5000} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    fireEvent.focus(input);

    // Input should still have value when focused
    expect(input).toHaveValue("5 seconds");
  });

  test("formats value on blur when valid", () => {
    render(<DurationInput value={0} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");

    // Type a shorthand duration
    fireEvent.change(input, { target: { value: "3h" } });
    expect(input).toHaveValue("3h");

    // Blur should format it
    fireEvent.blur(input);
    expect(input).toHaveValue("3 hours");
  });

  test("syncs back to value on blur when invalid", () => {
    render(<DurationInput value={5000} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");

    fireEvent.change(input, { target: { value: "invalid" } });
    expect(input).toHaveValue("invalid");

    fireEvent.blur(input);
    expect(input).toHaveValue("5 seconds");
  });

  test("does not format empty value on blur", () => {
    render(<DurationInput value={0} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");

    fireEvent.change(input, { target: { value: "" } });
    fireEvent.blur(input);

    expect(input).toHaveValue("");
  });

  test("updates input when value changes while not focused", () => {
    const { rerender } = render(
      <DurationInput value={5000} onChange={jest.fn()} />
    );

    const input = screen.getByRole("textbox");
    expect(input).toHaveValue("5 seconds");

    // Change value prop while not focused
    rerender(<DurationInput value={10800000} onChange={jest.fn()} />);

    expect(input).toHaveValue("3 hours");
  });

  test("does not update input when value changes while focused after local edit", () => {
    const { rerender } = render(
      <DurationInput value={5000} onChange={jest.fn()} />
    );

    const input = screen.getByRole("textbox");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "2h" } });

    // Change value prop while focused
    rerender(<DurationInput value={10800000} onChange={jest.fn()} />);

    // Should still show old value
    expect(input).toHaveValue("2h");
  });

  test("applies custom className", () => {
    const { container } = render(
      <DurationInput value={0} onChange={jest.fn()} className="custom-class" />
    );

    const wrapper = container.querySelector(".custom-class");
    expect(wrapper).toBeInTheDocument();
  });

  test("handles whitespace-only input as zero", () => {
    const onChange = jest.fn();
    render(<DurationInput value={5000} onChange={onChange} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "   " } });

    expect(onChange).toHaveBeenCalledWith(0);
  });

  test("handles exception from ms library", () => {
    // Mock ms to throw an error
    const originalMs = require("ms");
    require("ms").default = jest.fn(() => {
      throw new Error("Invalid format");
    });

    render(<DurationInput value={0} onChange={jest.fn()} />);

    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "invalid" } });

    expect(input).toHaveClass("invalid");

    // Restore original ms
    require("ms").default = originalMs.default;
  });
});
