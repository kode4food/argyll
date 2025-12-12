import React from "react";
import { render, screen } from "@testing-library/react";
import StepTypeLabel from "./StepTypeLabel";
import type { Step } from "../../api";
import { AttributeRole, AttributeType } from "../../api";

describe("StepTypeLabel", () => {
  const createStep = (hasInputs: boolean, hasOutputs: boolean): Step => ({
    id: "test-step",
    name: "Test Step",
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

  test("renders resolver label for step with outputs only", () => {
    const step = createStep(false, true);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByText("R")).toBeInTheDocument();
  });

  test("renders collector label for step with inputs only", () => {
    const step = createStep(true, false);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByText("C")).toBeInTheDocument();
  });

  test("renders processor label for step with inputs and outputs", () => {
    const step = createStep(true, true);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByText("P")).toBeInTheDocument();
  });

  test("renders neutral label for step with no inputs or outputs", () => {
    const step = createStep(false, false);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByText("S")).toBeInTheDocument();
  });

  test("applies correct CSS classes", () => {
    const step = createStep(true, true);
    const { container } = render(<StepTypeLabel step={step} />);
    const label = container.querySelector("span");
    expect(label?.className).toContain("step-type-label");
    expect(label?.className).toContain("processor");
  });
});
