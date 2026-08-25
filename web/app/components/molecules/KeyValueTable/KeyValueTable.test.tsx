import { fireEvent, render, screen } from "@testing-library/react";
import { IconAttributeLabel } from "@/utils/iconRegistry";
import KeyValueTable from "./KeyValueTable";

describe("KeyValueTable", () => {
  const baseProps = {
    addLabel: "Add Label",
    Icon: IconAttributeLabel,
    keyPlaceholder: "key",
    label: "Labels",
    onAdd: jest.fn(),
    onChange: jest.fn(),
    onRemove: jest.fn(),
    pairs: [{ id: "label-1", key: "team", value: "risk" }],
    removeLabel: "Remove Label",
    valuePlaceholder: "value",
    keySuggestions: ["team", "domain"],
    valueSuggestions: () => ["risk", "trading"],
  };

  beforeEach(() => jest.clearAllMocks());

  test("renders a row per pair", () => {
    render(<KeyValueTable {...baseProps} />);

    expect(screen.getByPlaceholderText("key")).toHaveValue("team");
    expect(screen.getByPlaceholderText("value")).toHaveValue("risk");
  });

  test("renders no rows without pairs", () => {
    render(<KeyValueTable {...baseProps} pairs={[]} />);

    expect(screen.queryByPlaceholderText("key")).not.toBeInTheDocument();
  });

  test("adds a pair", () => {
    render(<KeyValueTable {...baseProps} />);
    fireEvent.click(screen.getByRole("button", { name: "Add Label" }));

    expect(baseProps.onAdd).toHaveBeenCalled();
  });

  test("changes a key and a value", () => {
    render(<KeyValueTable {...baseProps} />);

    fireEvent.change(screen.getByPlaceholderText("key"), {
      target: { value: "domain" },
    });
    fireEvent.change(screen.getByPlaceholderText("value"), {
      target: { value: "trading" },
    });

    expect(baseProps.onChange).toHaveBeenNthCalledWith(
      1,
      "label-1",
      "key",
      "domain"
    );
    expect(baseProps.onChange).toHaveBeenNthCalledWith(
      2,
      "label-1",
      "value",
      "trading"
    );
  });

  test("removes a pair", () => {
    render(<KeyValueTable {...baseProps} />);
    fireEvent.click(screen.getByRole("button", { name: "Remove Label team" }));

    expect(baseProps.onRemove).toHaveBeenCalledWith("label-1");
  });
});
