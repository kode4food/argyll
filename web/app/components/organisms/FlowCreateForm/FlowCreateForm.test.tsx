import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from "@testing-library/react";
import FlowCreateForm from "./FlowCreateForm";
import styles from "./FlowCreateForm.module.css";
import { t } from "@/app/testUtils/i18n";
import { useUI } from "@/app/contexts/UIContext";
import { Step, AttributeRole, AttributeType } from "@/app/api";
import {
  FlowCreationContext,
  FlowCreationContextValue,
} from "@/app/contexts/FlowCreationContext";

jest.mock("@/app/contexts/UIContext");
jest.mock("@/app/hooks/useEscapeKey");
jest.mock("@/app/components/molecules/LazyCodeEditor", () => {
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

describe("FlowCreateForm", () => {
  const mockStep: Step = {
    id: "step-1",
    name: "Test Step",
    type: "sync",
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
    newID: "test-flow",
    setNewID: jest.fn(),
    setIDManuallyEdited: jest.fn(),
    handleStepChange: jest.fn(),
    initialState: "{}",
    setInitialState: jest.fn(),
    creating: false,
    handleCreateFlow: jest.fn(),
    steps: [mockStep],
    generateID: jest.fn(() => "generated-id"),
    sortSteps: jest.fn((steps: Step[]) => steps),
  };

  const defaultUIContext = {
    showCreateForm: true,
    setShowCreateForm: jest.fn(),
    disableEdit: false,
    diagramContainerRef: { current: null },
    previewPlan: null,
    setPreviewPlan: jest.fn(),
    goalSteps: [],
    toggleGoalStep: jest.fn(),
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
    setGoalSteps: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseUI.mockReturnValue(defaultUIContext);
  });

  const renderWithProvider = (
    overrides: Partial<FlowCreationContextValue> = {}
  ) => {
    const value: FlowCreationContextValue = {
      ...defaultProps,
      ...overrides,
    };
    return render(
      <FlowCreationContext.Provider value={value}>
        <FlowCreateForm />
      </FlowCreationContext.Provider>
    );
  };

  test("returns null when showCreateForm is false", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      showCreateForm: false,
    });

    const { container } = renderWithProvider();
    expect(container.firstChild).toBeNull();
  });

  test("renders form when showCreateForm is true", () => {
    const { container } = renderWithProvider();

    expect(container.querySelector(`.${styles.modal}`)).toBeInTheDocument();
    expect(container.querySelector(`.${styles.sidebar}`)).toBeInTheDocument();
    expect(container.querySelector(`.${styles.main}`)).toBeInTheDocument();
    expect(
      screen.getByText(t("flowCreate.selectGoalSteps"))
    ).toBeInTheDocument();
    expect(screen.getByText(t("flowCreate.flowIdLabel"))).toBeInTheDocument();
  });

  test("renders steps in sorted list", () => {
    const sortSteps = jest.fn((steps) =>
      [...steps].sort((a, b) => a.name.localeCompare(b.name))
    );

    const steps = [
      { ...mockStep, id: "step-1", name: "Zebra" },
      { ...mockStep, id: "step-2", name: "Alpha" },
    ];

    renderWithProvider({ steps, sortSteps });

    expect(sortSteps).toHaveBeenCalledWith(steps);
  });

  test("displays flow ID input with current value", () => {
    renderWithProvider({ newID: "my-flow" });

    const input = screen.getByPlaceholderText(
      t("flowCreate.flowIdPlaceholder")
    ) as HTMLInputElement;
    expect(input.value).toBe("my-flow");
  });

  test("calls setNewID and setIDManuallyEdited when ID input changes", () => {
    renderWithProvider();

    const input = screen.getByPlaceholderText("e.g., order-processing-001");
    fireEvent.change(input, { target: { value: "new-id" } });

    expect(defaultProps.setNewID).toHaveBeenCalledWith("new-id");
    expect(defaultProps.setIDManuallyEdited).toHaveBeenCalledWith(true);
  });

  test("generates new ID when generate button clicked", () => {
    renderWithProvider();

    const button = screen.getByLabelText("Generate new flow ID");
    fireEvent.click(button);

    expect(defaultProps.generateID).toHaveBeenCalled();
    expect(defaultProps.setNewID).toHaveBeenCalledWith("generated-id");
    expect(defaultProps.setIDManuallyEdited).toHaveBeenCalledWith(false);
  });

  test("displays initial state in code editor", () => {
    renderWithProvider({ initialState: '{"key": "value"}' });

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    const editor = screen.getByTestId("code-editor") as HTMLTextAreaElement;
    expect(editor.value).toBe('{"key": "value"}');
  });

  test("calls setInitialState when code editor changes", () => {
    renderWithProvider();

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    const editor = screen.getByTestId("code-editor");
    fireEvent.change(editor, { target: { value: '{"new": "value"}' } });

    expect(defaultProps.setInitialState).toHaveBeenCalledWith(
      '{"new": "value"}'
    );
  });

  test("shows JSON error when initialState is invalid JSON", () => {
    renderWithProvider({ initialState: "{invalid" });

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    expect(
      screen.getByText((content) =>
        content.startsWith(t("flowCreate.invalidJson", { error: "" }))
      )
    ).toBeInTheDocument();
  });

  test("does not show JSON error when initialState is valid JSON", () => {
    renderWithProvider({ initialState: '{"valid": true}' });

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    expect(
      screen.queryByText((content) =>
        content.startsWith(t("flowCreate.invalidJson", { error: "" }))
      )
    ).not.toBeInTheDocument();
  });

  test("closes form when overlay is clicked", () => {
    renderWithProvider();

    const overlay = screen.getByLabelText(t("flowCreate.closeForm"));
    fireEvent.click(overlay);

    expect(defaultUIContext.setShowCreateForm).toHaveBeenCalledWith(false);
  });

  test("closes form when Cancel button is clicked", () => {
    renderWithProvider();

    const cancelButton = screen.getByText(t("common.cancel"));
    fireEvent.click(cancelButton);

    expect(defaultUIContext.setShowCreateForm).toHaveBeenCalledWith(false);
  });

  test("calls handleCreateFlow when Start button is clicked", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    renderWithProvider({ newID: "test-id" });

    const startButton = screen.getByText(t("common.start"));
    fireEvent.click(startButton);

    expect(defaultProps.handleCreateFlow).toHaveBeenCalled();
  });

  test("disables Start button when creating", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    renderWithProvider({ newID: "test-id", creating: true });

    const startButton = screen.getByText(t("common.start"));
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when ID is empty", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    renderWithProvider({ newID: "" });

    const startButton = screen.getByText(t("common.start"));
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when no goal steps selected", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: [],
    });

    renderWithProvider({ newID: "test-id" });

    const startButton = screen.getByText(t("common.start"));
    expect(startButton).toBeDisabled();
  });

  test("disables Start button when JSON is invalid", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    renderWithProvider({ newID: "test-id", initialState: "{invalid" });
    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    const startButton = screen.getByText(t("common.start"));
    expect(startButton).toBeDisabled();
  });

  test("does not show Play icon when creating", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    const { container } = renderWithProvider({
      newID: "test-id",
      creating: true,
    });

    expect(container.querySelector(".lucide-play")).not.toBeInTheDocument();
  });

  test("shows warning when no steps are registered", () => {
    renderWithProvider({ steps: [] });

    expect(
      screen.getByText(t("flowCreate.warningNoSteps"))
    ).toBeInTheDocument();
  });

  test("does not show warning when steps are registered", () => {
    renderWithProvider({ steps: [mockStep] });

    expect(
      screen.queryByText(t("flowCreate.warningNoSteps"))
    ).not.toBeInTheDocument();
  });

  test("shows required attributes label only in JSON mode", () => {
    renderWithProvider();

    expect(
      screen.queryByText(t("flowCreate.requiredAttributesLabel"))
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );

    expect(
      screen.getByText(t("flowCreate.requiredAttributesLabel"))
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeBasic") })
    );

    expect(
      screen.queryByText(t("flowCreate.requiredAttributesLabel"))
    ).not.toBeInTheDocument();
  });

  test("selects step when clicked", async () => {
    renderWithProvider();

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).toHaveBeenCalledWith(["step-1"]);
    });
  });

  test("deselects step when already selected", async () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    renderWithProvider();

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).toHaveBeenCalledWith([]);
    });
  });

  test("marks step as selected with correct styling", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      goalSteps: ["step-1"],
    });

    const { container } = renderWithProvider();

    const stepItem = container.querySelector('[class*="dropdownItemSelected"]');
    expect(stepItem).toBeInTheDocument();
  });

  test("shows tooltip when step included in preview plan", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan: {
        steps: { "step-1": mockStep },
        attributes: {},
        goals: [],
        required: [],
      },
      goalSteps: [],
    });

    const { container } = renderWithProvider();

    const stepItem = container.querySelector(
      '[title="Already included in execution plan"]'
    );
    expect(stepItem).toBeInTheDocument();
  });

  test("shows tooltip when outputs satisfied by initial state", () => {
    renderWithProvider({ initialState: '{"output1": "value"}' });

    const stepItem = screen
      .getByText("Test Step")
      .closest('div[title="Outputs satisfied by initial state"]');
    expect(stepItem).toBeInTheDocument();
  });

  test("shows tooltip when step has missing requirements", () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan: {
        steps: {},
        attributes: {},
        goals: [],
        required: [],
        excluded: {
          missing: {
            "step-1": ["input1"],
          },
        },
      },
      goalSteps: [],
    });

    renderWithProvider();

    const stepItem = screen
      .getByText("Test Step")
      .closest('div[title="Missing required: input1"]');
    expect(stepItem).toBeInTheDocument();
  });

  test("does not trigger step change when disabled step clicked", async () => {
    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan: {
        steps: { "step-1": mockStep },
        attributes: {},
        goals: [],
        required: [],
      },
    });

    renderWithProvider();

    const stepItem = screen.getByText("Test Step").closest("div");
    fireEvent.click(stepItem!);

    await waitFor(() => {
      expect(defaultProps.handleStepChange).not.toHaveBeenCalled();
    });
  });

  test("marks required badge only for plan-required launch inputs", () => {
    const previewPlan = {
      goals: ["goal-step"],
      required: ["order_id"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync" as const,
          attributes: {
            order_id: {
              role: AttributeRole.Required,
              type: AttributeType.String,
            },
            quantity: {
              role: AttributeRole.Required,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        upstream: {
          id: "upstream",
          name: "Upstream",
          type: "sync" as const,
          attributes: {
            quantity: {
              role: AttributeRole.Output,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/upstream", timeout: 5000 },
        },
      },
    };

    mockUseUI.mockReturnValue({
      ...defaultUIContext,
      previewPlan,
      goalSteps: ["goal-step"],
    });

    renderWithProvider();

    const orderIdRow = screen
      .getByText("order_id")
      .closest(`.${styles.attributeListItem}`);
    const quantityRow = screen
      .getByText("quantity")
      .closest(`.${styles.attributeListItem}`);
    expect(orderIdRow).toBeInTheDocument();
    expect(quantityRow).toBeInTheDocument();

    expect(
      within(orderIdRow as HTMLElement).getByText(t("flowCreate.requiredBadge"))
    ).toBeInTheDocument();
    expect(
      within(quantityRow as HTMLElement).queryByText(
        t("flowCreate.requiredBadge")
      )
    ).not.toBeInTheDocument();
  });
});
