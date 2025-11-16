import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import WorkflowCreateForm from "./WorkflowCreateForm";
import { useUI } from "../../contexts/UIContext";
import { Step, AttributeRole, AttributeType } from "../../api";

jest.mock("../../contexts/UIContext");
jest.mock("../../hooks/useEscapeKey");
jest.mock("../molecules/LazyCodeEditor", () => {
  return function MockLazyCodeEditor({ value, onChange }: any) {
    return (
      <textarea
        data-testid="code-editor"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  };
});

const mockUseUI = useUI as jest.MockedFunction<typeof useUI>;

describe("WorkflowCreateForm", () => {
  const mockStep: Step = {
    id: "step-1",
    name: "Test Step",
    type: "sync",
    version: "1.0.0",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      output1: { role: AttributeRole.Output, type: AttributeType.String },
    },
    http: {
      endpoint: "http://localhost:8080/test",
      timeout: 5000,
    },
  };

  const defaultProps = {
    newID: "test-workflow",
    setNewID: jest.fn(),
    setIDManuallyEdited: jest.fn(),
    handleStepChange: jest.fn(),
    initialState: "{}",
    setInitialState: jest.fn(),
    creating: false,
    handleCreateWorkflow: jest.fn(),
    steps: [mockStep],
    generateID: jest.fn(() => "generated-id"),
    sortSteps: jest.fn((steps) => steps),
  };

  const defaultUIContext = {
    showCreateForm: true,
    setShowCreateForm: jest.fn(),
    disableEdit: false,
    diagramContainerRef: { current: null },
    previewPlan: null,
    selectedStep: null,
    goalStepIds: [],
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
    setSelectedStep: jest.fn(),
    setGoalStepIds: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseUI.mockReturnValue(defaultUIContext);
  });

  test("returns null when showCreateForm is false", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      showCreateForm: false,
    });

    const { container } = render(<WorkflowCreateForm {...defaultProps} />);
    expect(container.firstChild).toBeNull();
  });

  test("renders form when showCreateForm is true", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    expect(screen.getByText("Select Goal Steps")).toBeInTheDocument();
    expect(screen.getByText("Workflow ID")).toBeInTheDocument();
    expect(screen.getByText("Required Attributes")).toBeInTheDocument();
  });

  test("renders steps in sorted list", () => {
    const sortSteps = jest.fn((steps) =>
      [...steps].sort((a, b) => a.name.localeCompare(b.name))
    );

    const steps = [
      { ...mockStep, id: "step-1", name: "Zebra" },
      { ...mockStep, id: "step-2", name: "Alpha" },
    ];

    render(
      <WorkflowCreateForm
        {...defaultProps}
        steps={steps}
        sortSteps={sortSteps}
      />
    );

    expect(sortSteps).toHaveBeenCalledWith(steps);
  });

  test("displays workflow ID input with current value", () => {
    render(<WorkflowCreateForm {...defaultProps} newID="my-workflow" />);

    const input = screen.getByPlaceholderText(
      "e.g., order-processing-001"
    ) as HTMLInputElement;
    expect(input.value).toBe("my-workflow");
  });

  test("calls setNewID and setIDManuallyEdited when ID input changes", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const input = screen.getByPlaceholderText("e.g., order-processing-001");
    fireEvent.change(input, { target: { value: "new-id" } });

    expect(defaultProps.setNewID).toHaveBeenCalledWith("new-id");
    expect(defaultProps.setIDManuallyEdited).toHaveBeenCalledWith(true);
  });

  test("generates new ID when generate button clicked", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const button = screen.getByLabelText("Generate new workflow ID");
    fireEvent.click(button);

    expect(defaultProps.generateID).toHaveBeenCalled();
    expect(defaultProps.setNewID).toHaveBeenCalledWith("generated-id");
    expect(defaultProps.setIDManuallyEdited).toHaveBeenCalledWith(false);
  });

  test("displays initial state in code editor", () => {
    render(
      <WorkflowCreateForm {...defaultProps} initialState='{"key": "value"}' />
    );

    const editor = screen.getByTestId("code-editor") as HTMLTextAreaElement;
    expect(editor.value).toBe('{"key": "value"}');
  });

  test("calls setInitialState when code editor changes", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const editor = screen.getByTestId("code-editor");
    fireEvent.change(editor, { target: { value: '{"new": "value"}' } });

    expect(defaultProps.setInitialState).toHaveBeenCalledWith(
      '{"new": "value"}'
    );
  });

  test("shows JSON error when initialState is invalid JSON", () => {
    render(<WorkflowCreateForm {...defaultProps} initialState="{invalid" />);

    expect(screen.getByText(/Invalid JSON/)).toBeInTheDocument();
  });

  test("does not show JSON error when initialState is valid JSON", () => {
    render(
      <WorkflowCreateForm {...defaultProps} initialState='{"valid": true}' />
    );

    expect(screen.queryByText(/Invalid JSON/)).not.toBeInTheDocument();
  });

  test("closes form when overlay is clicked", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const overlay = screen.getByLabelText("Close workflow form");
    fireEvent.click(overlay);

    expect(defaultUIContext.setShowCreateForm).toHaveBeenCalledWith(false);
  });

  test("closes form when Cancel button is clicked", () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const cancelButton = screen.getByText("Cancel");
    fireEvent.click(cancelButton);

    expect(defaultUIContext.setShowCreateForm).toHaveBeenCalledWith(false);
  });

  test("calls handleCreateWorkflow when Start button is clicked", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    render(<WorkflowCreateForm {...defaultProps} newID="test-id" />);

    const startButton = screen.getByText("Start");
    fireEvent.click(startButton);

    expect(defaultProps.handleCreateWorkflow).toHaveBeenCalled();
  });

  test("disables Start button when creating", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    render(
      <WorkflowCreateForm {...defaultProps} newID="test-id" creating={true} />
    );

    const startButton = screen.getByText("Start");
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when ID is empty", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    render(<WorkflowCreateForm {...defaultProps} newID="" />);

    const startButton = screen.getByText("Start");
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when no goal steps selected", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: [],
    });

    render(<WorkflowCreateForm {...defaultProps} newID="test-id" />);

    const startButton = screen.getByText("Start");
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when JSON is invalid", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    render(
      <WorkflowCreateForm
        {...defaultProps}
        newID="test-id"
        initialState="{invalid"
      />
    );

    const startButton = screen.getByText("Start");
    expect(startButton).toBeDisabled();
  });

  test("does not show Play icon when creating", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    const { container } = render(
      <WorkflowCreateForm {...defaultProps} newID="test-id" creating={true} />
    );

    expect(container.querySelector(".lucide-play")).not.toBeInTheDocument();
  });

  test("shows warning when no steps are registered", () => {
    render(<WorkflowCreateForm {...defaultProps} steps={[]} />);

    expect(screen.getByText(/No steps are registered/)).toBeInTheDocument();
  });

  test("does not show warning when steps are registered", () => {
    render(<WorkflowCreateForm {...defaultProps} steps={[mockStep]} />);

    expect(
      screen.queryByText(/No steps are registered/)
    ).not.toBeInTheDocument();
  });

  test("selects step when clicked", async () => {
    render(<WorkflowCreateForm {...defaultProps} />);

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).toHaveBeenCalledWith(["step-1"]);
    });
  });

  test("deselects step when already selected", async () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    render(<WorkflowCreateForm {...defaultProps} />);

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).toHaveBeenCalledWith([]);
      expect(defaultUIContext.setSelectedStep).toHaveBeenCalledWith(null);
    });
  });

  test("marks step as selected with correct styling", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalStepIds: ["step-1"],
    });

    const { container } = render(<WorkflowCreateForm {...defaultProps} />);

    const stepItem = container.querySelector('[class*="dropdownItemSelected"]');
    expect(stepItem).toBeInTheDocument();
  });

  test("shows tooltip when step included in preview plan", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan: {
        steps: { "step-1": { step: mockStep } },
        attributes: {},
        goals: [],
        required: [],
      },
      goalStepIds: [],
    });

    const { container } = render(<WorkflowCreateForm {...defaultProps} />);

    const stepItem = container.querySelector(
      '[title="Already included in execution plan"]'
    );
    expect(stepItem).toBeInTheDocument();
  });

  test("shows tooltip when outputs satisfied by initial state", () => {
    render(
      <WorkflowCreateForm
        {...defaultProps}
        initialState='{"output1": "value"}'
      />
    );

    const stepItem = screen
      .getByText("Test Step")
      .closest('div[title="Outputs satisfied by initial state"]');
    expect(stepItem).toBeInTheDocument();
  });

  test("does not trigger step change when disabled step clicked", async () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan: {
        steps: { "step-1": { step: mockStep } },
        attributes: {},
        goals: [],
        required: [],
      },
    });

    render(<WorkflowCreateForm {...defaultProps} />);

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).not.toHaveBeenCalled();
    });
  });
});
