import React from "react";
import { render } from "@testing-library/react";
import Spinner from "./Spinner";

describe("Spinner", () => {
  test("renders with default props", () => {
    const { container } = render(<Spinner />);
    const spinner = container.querySelector("div");
    expect(spinner).toBeInTheDocument();
    expect(spinner?.className).toContain("animate-spin");
    expect(spinner?.className).toContain("h-8 w-8");
    expect(spinner?.className).toContain("border-blue-500");
  });

  test("renders with small size", () => {
    const { container } = render(<Spinner size="sm" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("h-4 w-4");
  });

  test("renders with medium size", () => {
    const { container } = render(<Spinner size="md" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("h-8 w-8");
  });

  test("renders with large size", () => {
    const { container } = render(<Spinner size="lg" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("h-12 w-12");
  });

  test("renders with blue color", () => {
    const { container } = render(<Spinner color="blue" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("border-blue-500");
  });

  test("renders with white color", () => {
    const { container } = render(<Spinner color="white" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("border-white");
  });

  test("applies custom className", () => {
    const { container } = render(<Spinner className="custom-spinner" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("custom-spinner");
  });

  test("combines size, color, and custom className", () => {
    const { container } = render(
      <Spinner size="lg" color="white" className="custom" />
    );
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain("h-12 w-12");
    expect(spinner?.className).toContain("border-white");
    expect(spinner?.className).toContain("custom");
  });
});
