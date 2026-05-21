import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import ComboInput from "./ComboInput";

describe("ComboInput", () => {
  const suggestions = ["alpha", "beta", "gamma"];
  const onChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders input with current value", () => {
    render(
      <ComboInput
        value="alpha"
        suggestions={suggestions}
        onChange={onChange}
        ariaLabel="test-combo"
      />
    );
    expect(screen.getByRole("textbox")).toHaveValue("alpha");
  });

  test("calls onChange when typing", () => {
    render(
      <ComboInput value="" suggestions={suggestions} onChange={onChange} />
    );
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "al" },
    });
    expect(onChange).toHaveBeenCalledWith("al");
  });

  test("opens suggestion list when trigger clicked", () => {
    render(
      <ComboInput value="" suggestions={suggestions} onChange={onChange} />
    );
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show suggestions" }));
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(3);
  });

  test("selects suggestion and closes list", () => {
    render(
      <ComboInput value="" suggestions={suggestions} onChange={onChange} />
    );
    fireEvent.click(screen.getByRole("button", { name: "Show suggestions" }));
    fireEvent.click(screen.getByRole("option", { name: "beta" }));
    expect(onChange).toHaveBeenCalledWith("beta");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("marks current value as selected in the list", () => {
    render(
      <ComboInput value="gamma" suggestions={suggestions} onChange={onChange} />
    );
    fireEvent.click(screen.getByRole("button", { name: "Show suggestions" }));
    const selected = screen.getByRole("option", { name: "gamma" });
    expect(selected).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("option", { name: "alpha" })).toHaveAttribute(
      "aria-selected",
      "false"
    );
  });

  test("closes list on outside mousedown", () => {
    render(
      <div>
        <ComboInput value="" suggestions={suggestions} onChange={onChange} />
        <button data-testid="outside">outside</button>
      </div>
    );
    fireEvent.click(screen.getByRole("button", { name: "Show suggestions" }));
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("toggle closes open list", () => {
    render(
      <ComboInput value="" suggestions={suggestions} onChange={onChange} />
    );
    const trigger = screen.getByRole("button", { name: "Show suggestions" });
    fireEvent.click(trigger);
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.click(trigger);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
