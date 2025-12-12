import React from "react";
import { render, screen } from "@testing-library/react";
import StepHeader from "./StepHeader";
import type { Step } from "../../api";
import { AttributeRole, AttributeType } from "../../api";

describe("StepHeader", () => {
  const createStep = (
    name: string,
    hasInputs: boolean,
    hasOutputs: boolean
  ): Step => ({
    id: "test-step",
    name,
    type: "sync",
    attributes: {
      ...(hasInputs
        ? {
            input: { role: AttributeRole.Required, type: AttributeType.String },
          }
        : {}),
      ...(hasOutputs
        ? { output: { role: AttributeRole.Output, type: AttributeType.String } }
        : {}),
    },
    version: "1.0.0",
    http: {
      endpoint: "http://test",
      timeout: 5000,
    },
  });

  test("renders step name", () => {
    const step = createStep("My Test Step", false, false);
    render(<StepHeader step={step} />);
    expect(
      screen.getByRole("heading", { name: "My Test Step" })
    ).toBeInTheDocument();
  });

  test("renders step name as h3", () => {
    const step = createStep("Test", false, false);
    render(<StepHeader step={step} />);
    const heading = screen.getByRole("heading", { name: "Test" });
    expect(heading.tagName).toBe("H3");
  });

  test("renders step name in heading", () => {
    const step = createStep("Long Step Name", false, false);
    render(<StepHeader step={step} />);
    const heading = screen.getByRole("heading", { name: "Long Step Name" });
    expect(heading).toBeInTheDocument();
  });

  test("renders StepTypeLabel for resolver", () => {
    const step = createStep("Resolver Step", false, true);
    render(<StepHeader step={step} />);
    expect(screen.getByText("R")).toBeInTheDocument();
  });

  test("renders StepTypeLabel for collector", () => {
    const step = createStep("Collector Step", true, false);
    render(<StepHeader step={step} />);
    expect(screen.getByText("C")).toBeInTheDocument();
  });

  test("renders StepTypeLabel for processor", () => {
    const step = createStep("Processor Step", true, true);
    render(<StepHeader step={step} />);
    expect(screen.getByText("P")).toBeInTheDocument();
  });

  test("applies correct CSS classes", () => {
    const step = createStep("Test", false, false);
    const { container } = render(<StepHeader step={step} />);
    const header = container.querySelector(".step-header");
    expect(header).toBeInTheDocument();
  });
});
