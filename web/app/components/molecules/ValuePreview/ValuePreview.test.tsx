import { fireEvent, render, screen } from "@testing-library/react";
import ValuePreview from "./ValuePreview";

const exact = { normalizer: (s: string) => s };

describe("ValuePreview", () => {
  test("shows compact JSON, and the full value on hover", () => {
    render(<ValuePreview title="Current Value" value={{ sum: 43 }} />);

    const preview = screen.getByText('{"sum":43}', exact);
    expect(preview.tagName).toBe("CODE");

    fireEvent.mouseEnter(preview);
    expect(screen.getByText('{\n  "sum": 43\n}', exact)).toBeInTheDocument();
    expect(screen.getByText("Current Value:")).toBeInTheDocument();
  });
});
