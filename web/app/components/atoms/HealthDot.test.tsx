import React from "react";
import { render } from "@testing-library/react";
import HealthDot from "./HealthDot";

describe("HealthDot", () => {
  test("renders with healthy status", () => {
    const { container } = render(<HealthDot status="healthy" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("healthy");
  });

  test("renders with unhealthy status", () => {
    const { container } = render(<HealthDot status="unhealthy" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unhealthy");
  });

  test("renders with unconfigured status", () => {
    const { container } = render(<HealthDot status="unconfigured" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unconfigured");
  });

  test("renders with unknown status", () => {
    const { container } = render(<HealthDot status="unknown" />);
    const dot = container.querySelector("div");
    expect(dot).toBeInTheDocument();
    expect(dot?.className).toContain("unknown");
  });
});
