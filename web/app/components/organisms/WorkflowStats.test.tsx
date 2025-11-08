import React from "react";
import { render, screen } from "@testing-library/react";
import WorkflowStats from "./WorkflowStats";
import type { Step } from "../../api";
import { AttributeRole, AttributeType } from "../../api";

describe("WorkflowStats", () => {
  const createStep = (
    id: string,
    requiredArgs: string[],
    optionalArgs: string[],
    outputs: string[]
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
    outputs.forEach((name) => {
      attributes[name] = {
        role: AttributeRole.Output,
        type: AttributeType.String,
      };
    });

    return {
      id,
      name: `Step ${id}`,
      type: "sync",
      attributes,
      version: "1.0.0",
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };
  };

  test("shows required inputs stat", () => {
    const steps = [createStep("step1", ["input1", "input2"], [], [])];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["input1"]}
      />
    );
    expect(screen.getByText(/1 of 2/)).toBeInTheDocument();
  });

  test("shows optional inputs stat", () => {
    const steps = [createStep("step1", [], ["opt1", "opt2"], [])];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["opt1"]}
      />
    );
    expect(screen.getByText(/1 of 2/)).toBeInTheDocument();
  });

  test("shows outputs stat", () => {
    const steps = [createStep("step1", [], [], ["out1", "out2"])];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["out1"]}
      />
    );
    expect(screen.getByText(/1 of 2/)).toBeInTheDocument();
  });

  test("aggregates stats from multiple steps", () => {
    const steps = [
      createStep("step1", ["in1"], [], ["out1"]),
      createStep("step2", ["in2"], [], ["out2"]),
    ];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1", "step2"]}
        resolvedAttributes={["in1", "in2", "out1"]}
      />
    );
    expect(screen.getByText(/2 of 2/)).toBeInTheDocument(); // required inputs
    expect(screen.getByText(/1 of 2/)).toBeInTheDocument(); // outputs
  });

  test("only includes steps in execution sequence", () => {
    const steps = [
      createStep("step1", ["in1"], [], []),
      createStep("step2", ["in2"], [], []),
    ];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["in1", "in2"]}
      />
    );
    expect(screen.getByText(/1 of 1/)).toBeInTheDocument();
  });

  test("hides stat when count is zero", () => {
    const steps = [createStep("step1", [], [], ["out1"])];
    const { container } = render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={[]}
      />
    );
    const badges = container.querySelectorAll(".stat-badge");
    expect(badges.length).toBe(1); // Only outputs badge
  });

  test("shows all three stat types", () => {
    const steps = [createStep("step1", ["in1"], ["opt1"], ["out1"])];
    const { container } = render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={[]}
      />
    );
    const badges = container.querySelectorAll(".stat-badge");
    expect(badges.length).toBe(3);
  });

  test("handles empty execution sequence", () => {
    const steps = [createStep("step1", ["in1"], [], ["out1"])];
    const { container } = render(
      <WorkflowStats
        steps={steps}
        executionSequence={[]}
        resolvedAttributes={[]}
      />
    );
    const badges = container.querySelectorAll(".stat-badge");
    expect(badges.length).toBe(0);
  });

  test("handles empty steps array", () => {
    const { container } = render(
      <WorkflowStats
        steps={[]}
        executionSequence={["step1"]}
        resolvedAttributes={[]}
      />
    );
    const badges = container.querySelectorAll(".stat-badge");
    expect(badges.length).toBe(0);
  });

  test("calculates resolved percentage correctly", () => {
    const steps = [createStep("step1", ["in1", "in2", "in3"], [], [])];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["in1", "in3"]}
      />
    );
    expect(screen.getByText(/2 of 3/)).toBeInTheDocument();
  });

  test("handles all resolved attributes", () => {
    const steps = [createStep("step1", ["in1"], ["opt1"], ["out1"])];
    render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={["in1", "opt1", "out1"]}
      />
    );
    const badges = screen.getAllByText(/1 of 1/);
    expect(badges.length).toBe(3);
  });

  test("applies correct CSS classes", () => {
    const steps = [createStep("step1", ["in1"], ["opt1"], ["out1"])];
    const { container } = render(
      <WorkflowStats
        steps={steps}
        executionSequence={["step1"]}
        resolvedAttributes={[]}
      />
    );
    expect(container.querySelector(".workflow-stats")).toBeInTheDocument();
    expect(
      container.querySelector(".stat-badge--required")
    ).toBeInTheDocument();
    expect(
      container.querySelector(".stat-badge--optional")
    ).toBeInTheDocument();
    expect(container.querySelector(".stat-badge--output")).toBeInTheDocument();
  });
});
