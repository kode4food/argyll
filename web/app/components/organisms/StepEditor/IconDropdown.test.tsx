import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import IconDropdown, { IconDropdownOption } from "./IconDropdown";

describe("IconDropdown", () => {
  const options: IconDropdownOption[] = [
    { value: "a", label: "Alpha", icon: <span>A</span> },
    { value: "b", label: "Beta", icon: <span>B</span> },
    { value: "c", label: "Gamma", icon: <span>C</span> },
  ];
  const onChange = jest.fn();
  const faceIcon = <span>icon</span>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders face button without list", () => {
    render(
      <IconDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value="a"
        options={options}
        onChange={onChange}
      />
    );
    expect(
      screen.getByRole("button", { name: "choose role" })
    ).toBeInTheDocument();
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("opens list on button click", () => {
    render(
      <IconDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value="a"
        options={options}
        onChange={onChange}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(3);
  });

  test("marks current value as selected", () => {
    render(
      <IconDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value="b"
        options={options}
        onChange={onChange}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    expect(screen.getByRole("option", { name: /Beta/ })).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(screen.getByRole("option", { name: /Alpha/ })).toHaveAttribute(
      "aria-selected",
      "false"
    );
  });

  test("calls onChange and closes on option click", () => {
    render(
      <IconDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value="a"
        options={options}
        onChange={onChange}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "choose role" }));
    fireEvent.click(screen.getByRole("option", { name: /Gamma/ }));
    expect(onChange).toHaveBeenCalledWith("c");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("closes on outside mousedown", () => {
    render(
      <div>
        <IconDropdown
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
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("toggle closes open list", () => {
    render(
      <IconDropdown
        ariaLabel="choose role"
        faceIcon={faceIcon}
        value="a"
        options={options}
        onChange={onChange}
      />
    );
    const btn = screen.getByRole("button", { name: "choose role" });
    fireEvent.click(btn);
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.click(btn);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
