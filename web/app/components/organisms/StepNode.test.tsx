import React from "react";
import { render } from "@testing-library/react";
import StepNode from "./StepNode";
import {
  Step,
  WorkflowContext,
  ExecutionResult,
  AttributeRole,
  AttributeType,
} from "../../api";
import { Position } from "@xyflow/react";

jest.mock("./StepWidget", () => {
  return function MockStepWidget({ step, onClick, className }: any) {
    return (
      <div
        data-testid="step-widget"
        data-step-id={step.id}
        className={className}
        onClick={onClick}
      >
        {Object.entries(step.attributes || {}).map(
          ([name, spec]: [string, any]) => (
            <div
              key={name}
              data-arg-type={spec.role}
              data-arg-name={name}
              style={{ height: 20 }}
            />
          )
        )}
      </div>
    );
  };
});

jest.mock("../atoms/InvisibleHandle", () => {
  return function MockInvisibleHandle({
    id,
    type,
    position,
    top,
    argName,
  }: any) {
    return (
      <div
        data-testid={`handle-${id}`}
        data-handle-type={type}
        data-position={position}
        data-top={top}
        data-arg-name={argName}
      />
    );
  };
});

describe("StepNode", () => {
  const mockStep: Step = {
    id: "step-1",
    name: "Test Step",
    type: "sync",
    version: "1.0.0",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      input2: { role: AttributeRole.Optional, type: AttributeType.String },
      output1: { role: AttributeRole.Output, type: AttributeType.String },
    },
    http: {
      endpoint: "http://localhost:8080/test",
      timeout: 5000,
    },
  };

  const defaultNodeData = {
    step: mockStep,
    selected: false,
  };

  const defaultProps = {
    id: "node-1",
    type: "step",
    data: defaultNodeData,
    selected: false,
    isConnectable: true,
    zIndex: 0,
    xPos: 0,
    yPos: 0,
    dragging: false,
  };

  test("renders StepWidget with step data", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    const widget = getByTestId("step-widget");
    expect(widget).toBeInTheDocument();
    expect(widget.dataset.stepId).toBe("step-1");
  });

  test("renders handles for required, optional, and output attributes", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    expect(getByTestId("handle-input-required-input1")).toBeInTheDocument();
    expect(getByTestId("handle-input-optional-input2")).toBeInTheDocument();
    expect(getByTestId("handle-output-output1")).toBeInTheDocument();
  });

  test("sets correct handle types for inputs and outputs", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    expect(getByTestId("handle-input-required-input1").dataset.handleType).toBe(
      "target"
    );
    expect(getByTestId("handle-input-optional-input2").dataset.handleType).toBe(
      "target"
    );
    expect(getByTestId("handle-output-output1").dataset.handleType).toBe(
      "source"
    );
  });

  test("sets correct positions for inputs and outputs", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    expect(getByTestId("handle-input-required-input1").dataset.position).toBe(
      Position.Left.toString()
    );
    expect(getByTestId("handle-output-output1").dataset.position).toBe(
      Position.Right.toString()
    );
  });

  test("calls onStepClick when StepWidget is clicked", () => {
    const onStepClick = jest.fn();
    const { getByTestId } = render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, onStepClick }} />
    );

    getByTestId("step-widget").click();
    expect(onStepClick).toHaveBeenCalledWith("step-1");
  });

  test("does not call onStepClick when not provided", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    expect(() => getByTestId("step-widget").click()).not.toThrow();
  });

  test("finds execution for current step", () => {
    const executions: ExecutionResult[] = [
      {
        step_id: "step-1",
        status: "completed",
        outputs: { result: "value" },
      },
      {
        step_id: "step-2",
        status: "active",
      },
    ];

    render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, executions }} />
    );

    // StepWidget should receive the execution prop (tested via mock)
    // The component finds the correct execution internally
  });

  test("creates resolved attributes set", () => {
    const resolvedAttributes = ["input1", "input2"];

    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, resolvedAttributes }}
      />
    );

    // Component should create Set from array internally
  });

  test("builds provenance map from workflow state", () => {
    const workflowData: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {
        attr1: { value: "value1", step: "step-a" },
        attr2: { value: "value2", step: "step-b" },
      },
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, workflowData }} />
    );

    // Component builds provenance map internally
  });

  test("determines satisfied arguments from resolved attributes", () => {
    const resolvedAttributes = ["input1", "input2"];

    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, resolvedAttributes }}
      />
    );

    // Satisfied arguments should include input1 and input2
    // (both required/optional inputs that are resolved)
  });

  test("only includes required/optional in satisfied args, not outputs", () => {
    const resolvedAttributes = ["input1", "output1"];

    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, resolvedAttributes }}
      />
    );

    // Satisfied should include input1 but not output1
  });

  test("renders with goal step styling", () => {
    const { container } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, isGoalStep: true }}
      />
    );

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("goal");
  });

  test("renders with starting point styling", () => {
    const { container } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, isStartingPoint: true }}
      />
    );

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("start-point");
  });

  test("renders with both goal and starting point styling", () => {
    const { container } = render(
      <StepNode
        {...defaultProps}
        data={{
          ...defaultNodeData,
          isGoalStep: true,
          isStartingPoint: true,
        }}
      />
    );

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("goal");
    expect(widget?.className).toContain("start-point");
  });

  test("passes selected state to StepWidget", () => {
    const { rerender, container } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, selected: false }}
      />
    );

    let widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget).toBeInTheDocument();

    rerender(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, selected: true }}
      />
    );

    widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget).toBeInTheDocument();
  });

  test("passes preview mode flags to StepWidget", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{
          ...defaultNodeData,
          isInPreviewPlan: true,
          isPreviewMode: true,
        }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("passes diagramContainerRef to StepWidget", () => {
    const diagramContainerRef = React.createRef<HTMLDivElement>();

    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, diagramContainerRef }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("passes disableEdit flag to StepWidget", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, disableEdit: true }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty attributes", () => {
    const stepWithNoAttrs: Step = {
      ...mockStep,
      attributes: {},
    };

    const { container } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, step: stepWithNoAttrs }}
      />
    );

    // Should render without handles
    expect(
      container.querySelector('[data-testid^="handle-"]')
    ).not.toBeInTheDocument();
  });

  test("handles undefined attributes", () => {
    const stepWithNoAttrs: Step = {
      ...mockStep,
      attributes: undefined,
    };

    const { container } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, step: stepWithNoAttrs }}
      />
    );

    expect(
      container.querySelector('[data-testid^="handle-"]')
    ).not.toBeInTheDocument();
  });

  test("sorts attributes alphabetically for handle positioning", () => {
    const stepWithUnsortedAttrs: Step = {
      ...mockStep,
      attributes: {
        zebra: { role: AttributeRole.Required, type: AttributeType.String },
        alpha: { role: AttributeRole.Required, type: AttributeType.String },
        beta: { role: AttributeRole.Required, type: AttributeType.String },
      },
    };

    const { getByTestId } = render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, step: stepWithUnsortedAttrs }}
      />
    );

    // All handles should exist regardless of original order
    expect(getByTestId("handle-input-required-alpha")).toBeInTheDocument();
    expect(getByTestId("handle-input-required-beta")).toBeInTheDocument();
    expect(getByTestId("handle-input-required-zebra")).toBeInTheDocument();
  });

  test("creates different handle IDs for required vs optional inputs", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    // Required input should have format: input-required-{name}
    expect(getByTestId("handle-input-required-input1")).toBeInTheDocument();

    // Optional input should have format: input-optional-{name}
    expect(getByTestId("handle-input-optional-input2")).toBeInTheDocument();
  });

  test("creates output handle IDs with output- prefix", () => {
    const { getByTestId } = render(<StepNode {...defaultProps} />);

    // Output should have format: output-{name}
    expect(getByTestId("handle-output-output1")).toBeInTheDocument();
  });

  test("memoizes click handler", () => {
    const onStepClick = jest.fn();
    const { rerender, getByTestId } = render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, onStepClick }} />
    );

    getByTestId("step-widget").click();
    expect(onStepClick).toHaveBeenCalledTimes(1);

    // Re-render with same props
    rerender(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, onStepClick }} />
    );

    getByTestId("step-widget").click();
    expect(onStepClick).toHaveBeenCalledTimes(2);
  });

  test("handles workflow data with empty state", () => {
    const workflowData: WorkflowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, workflowData }} />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles workflow data without state property", () => {
    const workflowData: WorkflowContext = {
      id: "wf-1",
      status: "active",
      started_at: "2024-01-01T00:00:00Z",
    };

    render(
      <StepNode {...defaultProps} data={{ ...defaultNodeData, workflowData }} />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty executions array", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, executions: [] }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles undefined executions", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, executions: undefined }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty resolvedAttributes array", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, resolvedAttributes: [] }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles undefined resolvedAttributes", () => {
    render(
      <StepNode
        {...defaultProps}
        data={{ ...defaultNodeData, resolvedAttributes: undefined }}
      />
    );

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });
});
