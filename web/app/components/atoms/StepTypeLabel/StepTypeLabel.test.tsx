import { render, screen } from "@testing-library/react";
import StepTypeLabel from "./StepTypeLabel";
import type { Step } from "@/app/api";
import { AttributeRole, AttributeType } from "@/app/api";

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
    http: {
      endpoint: "http://test",
      timeout: 5000,
    },
  });

  test("renders sync icon inside resolver badge for step with outputs only", () => {
    const step = createStep(false, true);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByLabelText("sync")).toBeInTheDocument();
  });

  test("renders sync icon inside collector badge for step with inputs only", () => {
    const step = createStep(true, false);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByLabelText("sync")).toBeInTheDocument();
  });

  test("renders sync icon inside processor badge for step with inputs and outputs", () => {
    const step = createStep(true, true);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByLabelText("sync")).toBeInTheDocument();
  });

  test("renders sync icon inside standalone badge for step with no inputs or outputs", () => {
    const step = createStep(false, false);
    render(<StepTypeLabel step={step} />);
    expect(screen.getByLabelText("sync")).toBeInTheDocument();
  });

  test("renders flow icon for flow steps", () => {
    const step: Step = {
      id: "flow-step",
      name: "Flow Step",
      type: "flow",
      attributes: {},
      flow: {
        goals: ["goal-a"],
      },
    };

    render(<StepTypeLabel step={step} />);

    expect(screen.getByLabelText("flow")).toBeInTheDocument();
  });

  test("applies correct CSS classes", () => {
    const step = createStep(true, true);
    const { container } = render(<StepTypeLabel step={step} />);
    const label = container.querySelector("span");
    expect(label?.className).toContain("step-type-label");
    expect(label?.className).toContain("processor");
  });
});
