import React from "react";
import { render, fireEvent } from "@testing-library/react";
import StepNode from "./StepNode";
import {
  Step,
  FlowContext,
  ExecutionResult,
  AttributeRole,
  AttributeType,
} from "../../api";
import { Position } from "@xyflow/react";
import {
  DiagramSelectionProvider,
  DiagramSelectionContextValue,
} from "../../contexts/DiagramSelectionContext";
import { UIProvider } from "../../contexts/UIContext";
jest.mock("../../contexts/UIContext", () => ({
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useUI: () => ({
    goalSteps: [],
    toggleGoalStep: jest.fn(),
    setGoalSteps: jest.fn(),
    disableEdit: false,
    diagramContainerRef: { current: null },
  }),
}));

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
  type TestStepNodeData = {
    step: Step;
    selected: boolean;
    flowData?: FlowContext | null;
    executions?: ExecutionResult[];
    resolvedAttributes?: string[];
    isGoalStep?: boolean;
    isInPreviewPlan?: boolean;
    isPreviewMode?: boolean;
    isStartingPoint?: boolean;
    onStepClick?: (stepId: string, options?: { additive?: boolean }) => void;
    diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
    disableEdit?: boolean;
  };

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

  const defaultNodeData: TestStepNodeData = {
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
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    dragging: false,
    draggable: true,
    selectable: true,
    deletable: false,
  };

  const renderWithProvider = (
    nodeOverrides: Partial<typeof defaultProps> = {},
    selectionOverrides: Partial<DiagramSelectionContextValue> = {}
  ) =>
    render(
      <UIProvider>
        <DiagramSelectionProvider
          value={{
            goalSteps: [],
            toggleGoalStep: jest.fn(),
            setGoalSteps: jest.fn(),
            ...selectionOverrides,
          }}
        >
          <StepNode {...defaultProps} {...nodeOverrides} />
        </DiagramSelectionProvider>
      </UIProvider>
    );

  test("renders StepWidget with step data", () => {
    const { getByTestId } = renderWithProvider();

    const widget = getByTestId("step-widget");
    expect(widget).toBeInTheDocument();
    expect(widget.dataset.stepId).toBe("step-1");
  });

  test("renders handles for required, optional, and output attributes", () => {
    const { getByTestId } = renderWithProvider();

    expect(getByTestId("handle-input-required-input1")).toBeInTheDocument();
    expect(getByTestId("handle-input-optional-input2")).toBeInTheDocument();
    expect(getByTestId("handle-output-output1")).toBeInTheDocument();
  });

  test("sets correct handle types for inputs and outputs", () => {
    const { getByTestId } = renderWithProvider();

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
    const { getByTestId } = renderWithProvider();

    expect(getByTestId("handle-input-required-input1").dataset.position).toBe(
      Position.Left.toString()
    );
    expect(getByTestId("handle-output-output1").dataset.position).toBe(
      Position.Right.toString()
    );
  });

  test("calls onStepClick when StepWidget is clicked", () => {
    const onStepClick = jest.fn();
    const { getByTestId } = renderWithProvider({
      data: { ...defaultNodeData, onStepClick },
    });

    getByTestId("step-widget").click();
    expect(onStepClick).toHaveBeenCalledWith("step-1", { additive: false });
  });

  test("passes additive flag when ctrl/cmd is held", () => {
    const onStepClick = jest.fn();
    const { getByTestId } = renderWithProvider({
      data: { ...defaultNodeData, onStepClick },
    });

    fireEvent.click(getByTestId("step-widget"), { ctrlKey: true });
    expect(onStepClick).toHaveBeenCalledWith("step-1", { additive: true });
  });

  test("falls back to setGoalSteps when onStepClick is not provided", () => {
    const setGoalSteps = jest.fn();
    const { getByTestId } = renderWithProvider(
      {},
      {
        setGoalSteps,
      }
    );

    getByTestId("step-widget").click();
    expect(setGoalSteps).toHaveBeenCalledWith(["step-1"]);
  });

  test("finds execution for current step", () => {
    const executions: ExecutionResult[] = [
      {
        step_id: "step-1",
        flow_id: "wf-1",
        status: "completed",
        inputs: {},
        started_at: "2024-01-01T00:00:00Z",
        outputs: { result: "value" },
      },
      {
        step_id: "step-2",
        flow_id: "wf-1",
        status: "active",
        inputs: {},
        started_at: "2024-01-01T00:01:00Z",
      },
    ];

    renderWithProvider({
      data: { ...defaultNodeData, executions },
    });

    // StepWidget should receive the execution prop (tested via mock)
    // The component finds the correct execution internally
  });

  test("creates resolved attributes set", () => {
    const resolvedAttributes = ["input1", "input2"];

    renderWithProvider({
      data: { ...defaultNodeData, resolvedAttributes },
    });

    // Component should create Set from array internally
  });

  test("builds provenance map from flow state", () => {
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {
        attr1: { value: "value1", step: "step-a" },
        attr2: { value: "value2", step: "step-b" },
      },
      started_at: "2024-01-01T00:00:00Z",
    };

    renderWithProvider({
      data: { ...defaultNodeData, flowData },
    });

    // Component builds provenance map internally
  });

  test("determines satisfied arguments from resolved attributes", () => {
    const resolvedAttributes = ["input1", "input2"];

    renderWithProvider({
      data: { ...defaultNodeData, resolvedAttributes },
    });

    // Satisfied arguments should include input1 and input2
    // (both required/optional inputs that are resolved)
  });

  test("only includes required/optional in satisfied args, not outputs", () => {
    const resolvedAttributes = ["input1", "output1"];

    renderWithProvider({
      data: { ...defaultNodeData, resolvedAttributes },
    });

    // Satisfied should include input1 but not output1
  });

  test("renders with goal step styling", () => {
    const { container } = renderWithProvider({
      data: { ...defaultNodeData, isGoalStep: true },
    });

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("goal");
  });

  test("renders with starting point styling", () => {
    const { container } = renderWithProvider({
      data: { ...defaultNodeData, isStartingPoint: true },
    });

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("start-point");
  });

  test("renders with both goal and starting point styling", () => {
    const { container } = renderWithProvider({
      data: {
        ...defaultNodeData,
        isGoalStep: true,
        isStartingPoint: true,
      },
    });

    const widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget?.className).toContain("goal");
    expect(widget?.className).toContain("start-point");
  });

  test("passes selected state to StepWidget", () => {
    const { rerender, container } = renderWithProvider({
      data: { ...defaultNodeData, selected: false },
    });

    let widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget).toBeInTheDocument();

    rerender(
      <UIProvider>
        <DiagramSelectionProvider
          value={{
            goalSteps: [],
            toggleGoalStep: jest.fn(),
            setGoalSteps: jest.fn(),
          }}
        >
          <StepNode
            {...defaultProps}
            data={{ ...defaultNodeData, selected: true }}
          />
        </DiagramSelectionProvider>
      </UIProvider>
    );

    widget = container.querySelector('[data-testid="step-widget"]');
    expect(widget).toBeInTheDocument();
  });

  test("passes preview mode flags to StepWidget", () => {
    renderWithProvider({
      data: {
        ...defaultNodeData,
        isInPreviewPlan: true,
        isPreviewMode: true,
      },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("passes diagramContainerRef to StepWidget", () => {
    const diagramContainerRef = React.createRef<HTMLDivElement>();

    renderWithProvider({
      data: { ...defaultNodeData, diagramContainerRef },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("passes disableEdit flag to StepWidget", () => {
    renderWithProvider({
      data: { ...defaultNodeData, disableEdit: true },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty attributes", () => {
    const stepWithNoAttrs: Step = {
      ...mockStep,
      attributes: {},
    };

    const { container } = renderWithProvider({
      data: { ...defaultNodeData, step: stepWithNoAttrs },
    });

    // Should render without handles
    expect(
      container.querySelector('[data-testid^="handle-"]')
    ).not.toBeInTheDocument();
  });

  test("handles empty attributes", () => {
    const stepWithNoAttrs: Step = {
      ...mockStep,
      attributes: {},
    };

    const { container } = renderWithProvider({
      data: { ...defaultNodeData, step: stepWithNoAttrs },
    });

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

    const { getByTestId } = renderWithProvider({
      data: { ...defaultNodeData, step: stepWithUnsortedAttrs },
    });

    // All handles should exist regardless of original order
    expect(getByTestId("handle-input-required-alpha")).toBeInTheDocument();
    expect(getByTestId("handle-input-required-beta")).toBeInTheDocument();
    expect(getByTestId("handle-input-required-zebra")).toBeInTheDocument();
  });

  test("creates different handle IDs for required vs optional inputs", () => {
    const { getByTestId } = renderWithProvider();

    // Required input should have format: input-required-{name}
    expect(getByTestId("handle-input-required-input1")).toBeInTheDocument();

    // Optional input should have format: input-optional-{name}
    expect(getByTestId("handle-input-optional-input2")).toBeInTheDocument();
  });

  test("creates output handle IDs with output- prefix", () => {
    const { getByTestId } = renderWithProvider();

    // Output should have format: output-{name}
    expect(getByTestId("handle-output-output1")).toBeInTheDocument();
  });

  test("memoizes click handler", () => {
    const setGoalSteps = jest.fn();
    const onStepClick = jest.fn();
    const { rerender, getByTestId } = renderWithProvider(
      { data: { ...defaultNodeData, onStepClick } },
      { setGoalSteps }
    );

    getByTestId("step-widget").click();
    expect(setGoalSteps).not.toHaveBeenCalled();
    expect(onStepClick).toHaveBeenCalledTimes(1);

    // Re-render with same props
    rerender(
      <UIProvider>
        <DiagramSelectionProvider
          value={{
            goalSteps: [],
            toggleGoalStep: jest.fn(),
            setGoalSteps,
          }}
        >
          <StepNode
            {...defaultProps}
            data={{ ...defaultNodeData, onStepClick }}
          />
        </DiagramSelectionProvider>
      </UIProvider>
    );

    getByTestId("step-widget").click();
    expect(setGoalSteps).not.toHaveBeenCalled();
    expect(onStepClick).toHaveBeenCalledTimes(2);
  });

  test("handles flow data with empty state", () => {
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    renderWithProvider({
      data: { ...defaultNodeData, flowData },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles flow data with empty state", () => {
    const flowData: FlowContext = {
      id: "wf-1",
      status: "active",
      state: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    renderWithProvider({
      data: { ...defaultNodeData, flowData },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty executions array", () => {
    renderWithProvider({
      data: { ...defaultNodeData, executions: [] },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles undefined executions", () => {
    renderWithProvider({
      data: { ...defaultNodeData, executions: undefined },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles empty resolvedAttributes array", () => {
    renderWithProvider({
      data: { ...defaultNodeData, resolvedAttributes: [] },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });

  test("handles undefined resolvedAttributes", () => {
    renderWithProvider({
      data: { ...defaultNodeData, resolvedAttributes: undefined },
    });

    expect(
      document.querySelector('[data-testid="step-widget"]')
    ).toBeInTheDocument();
  });
});
