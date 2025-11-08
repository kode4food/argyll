import React from "react";
import { render } from "@testing-library/react";
import HealthDot from "./HealthDot";

describe("HealthDot", () => {
  test("renders with healthy className", () => {
    const { container } = render(<HealthDot className="healthy" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("healthy");
  });

  test("renders with unhealthy className", () => {
    const { container } = render(<HealthDot className="unhealthy" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unhealthy");
  });

  test("renders with unconfigured className", () => {
    const { container } = render(<HealthDot className="unconfigured" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unconfigured");
  });

  test("renders with unknown className", () => {
    const { container } = render(<HealthDot className="unknown" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unknown");
  });
});
