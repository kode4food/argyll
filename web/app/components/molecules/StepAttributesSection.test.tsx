import React from "react";
import { render, screen } from "@testing-library/react";
import StepAttributesSection from "./StepAttributesSection";
import type { Step, ExecutionResult } from "../../api";
import { AttributeRole, AttributeType } from "../../api";

jest.mock("../atoms/Tooltip", () => ({
  __esModule: true,
  default: ({ trigger, children }: any) => (
    <div data-testid="tooltip">
      {trigger}
      <div data-testid="tooltip-content">{children}</div>
    </div>
  ),
}));

describe("StepAttributesSection", () => {
  const createStep = (
    requiredArgs: string[],
    optionalArgs: string[],
    outputArgs: string[]
  ): Step => {
    const attributes: Record<string, any> = {};
    requiredArgs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Required,
        type: AttributeType.String,
      };
    });
    optionalArgs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Optional,
        type: AttributeType.String,
      };
    });
    outputArgs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Output,
        type: AttributeType.String,
      };
    });

    return {
      id: "step-1",
      name: "Test Step",
      type: "sync",
      attributes,
      version: "1.0.0",
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };
  };

  test("renders attributes section with required args", () => {
    const step = createStep(["input1", "input2"], [], []);
    const satisfiedArgs = new Set<string>();

    const { container } = render(
      <StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />
    );

    expect(
      container.querySelector('[data-arg-name="input1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="input2"]')
    ).toBeInTheDocument();
  });

  test("renders attributes section with optional args", () => {
    const step = createStep([], ["opt1", "opt2"], []);
    const satisfiedArgs = new Set<string>();

    const { container } = render(
      <StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />
    );

    expect(
      container.querySelector('[data-arg-name="opt1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="opt2"]')
    ).toBeInTheDocument();
  });

  test("renders all attributes in single section", () => {
    const step = createStep(["req1"], ["opt1"], ["out1"]);
    const satisfiedArgs = new Set<string>();

    const { container } = render(
      <StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />
    );

    expect(
      container.querySelector('[data-arg-name="req1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="opt1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="out1"]')
    ).toBeInTheDocument();
  });

  test("renders output attributes", () => {
    const step = createStep([], [], ["out1", "out2"]);
    const satisfiedArgs = new Set<string>();

    const { container } = render(
      <StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />
    );

    expect(
      container.querySelector('[data-arg-name="out1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="out2"]')
    ).toBeInTheDocument();
  });

  test("shows status badges when showStatus is true", () => {
    const step = createStep(["input1", "input2"], [], []);
    const satisfiedArgs = new Set(["input1"]);

    const { container } = render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        showStatus
      />
    );

    const badges = container.querySelectorAll(".arg-status-badge");
    expect(badges.length).toBeGreaterThan(0);
  });

  test("renders execution input values in tooltip", () => {
    const step = createStep(["input1"], [], []);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: { input1: "test value" },
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
      />
    );

    expect(screen.getByText(/"test value"/)).toBeInTheDocument();
  });

  test("renders execution output values in tooltip", () => {
    const step = createStep([], [], ["result"]);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {},
      outputs: { result: "output value" },
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
      />
    );

    expect(screen.getByText(/"output value"/)).toBeInTheDocument();
  });

  test("formats different value types correctly", () => {
    const step = createStep(["str", "num", "obj", "nullVal"], [], []);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {
        str: "string",
        num: 42,
        obj: { key: "value" },
        nullVal: null,
      },
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
      />
    );

    expect(screen.getByText(/"string"/)).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("null")).toBeInTheDocument();
  });

  test("displays timeout for optional args", () => {
    const step: Step = {
      id: "step-1",
      name: "Test",
      type: "sync",
      attributes: {
        opt1: { role: AttributeRole.Optional, type: AttributeType.String },
      },
      version: "1.0.0",
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };
    const satisfiedArgs = new Set<string>();

    render(<StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />);

    // Test removed - step-level timeout no longer shown per-argument
  });

  test("renders nothing when step has no args", () => {
    const step = createStep([], [], []);
    const satisfiedArgs = new Set<string>();

    const { container } = render(
      <StepAttributesSection step={step} satisfiedArgs={satisfiedArgs} />
    );

    expect(
      container.querySelector(".step-args-section")
    ).not.toBeInTheDocument();
  });

  test("handles execution with partial input args", () => {
    const step = createStep(["input1", "input2"], [], []);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "active",
      inputs: { input1: "value1" },
      started_at: "2024-01-01T00:00:00Z",
    };

    const { container } = render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
      />
    );

    expect(screen.getByText(/"value1"/)).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="input2"]')
    ).toBeInTheDocument();
  });

  test("formats complex objects in tooltips", () => {
    const step = createStep(["config"], [], []);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {
        config: { nested: { key: "value" }, array: [1, 2, 3] },
      },
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
      />
    );

    expect(screen.getByText(/"nested"/)).toBeInTheDocument();
    expect(screen.getByText(/"array"/)).toBeInTheDocument();
  });

  test("shows attributeProvenance for outputs", () => {
    const step = createStep([], [], ["result"]);
    const satisfiedArgs = new Set<string>();
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {},
      outputs: { result: 42 },
      started_at: "2024-01-01T00:00:00Z",
    };
    const attributeProvenance = new Map([["result", "step-1"]]);

    const { container } = render(
      <StepAttributesSection
        step={step}
        satisfiedArgs={satisfiedArgs}
        execution={execution}
        attributeProvenance={attributeProvenance}
        showStatus
      />
    );

    const winnerBadge = container.querySelector(".arg-status-badge.satisfied");
    expect(winnerBadge).toBeInTheDocument();
    expect(winnerBadge?.querySelector(".lucide-award")).toBeInTheDocument();
  });
});
