import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AttributeRole, AttributeType, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorFlowConfiguration from "./StepEditorFlowConfiguration";

const mockApplyFlowGoalSelectionChange = jest.fn();
const mockUseFlowFormStepFiltering = jest.fn();

jest.mock("@/utils/flowGoalSelectionModel", () => ({
  applyFlowGoalSelectionChange: (...args: any[]) =>
    mockApplyFlowGoalSelectionChange(...args),
}));

jest.mock("../FlowCreateForm/useFlowFormStepFiltering", () => ({
  useFlowFormStepFiltering: (...args: any[]) =>
    mockUseFlowFormStepFiltering(...args),
}));

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: {
    getExecutionPlan: jest.fn(),
  },
}));

describe("StepEditorFlowConfiguration", () => {
  const steps: Step[] = [
    {
      id: "current-step",
      name: "Current Step",
      type: "flow",
      attributes: {},
      flow: { goals: [] },
    },
    {
      id: "alpha",
      name: "Alpha",
      type: "sync",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
      },
      http: { endpoint: "http://localhost/a", timeout: 5000 },
    },
    {
      id: "beta",
      name: "Beta",
      type: "sync",
      attributes: {},
      http: { endpoint: "http://localhost/b", timeout: 5000 },
    },
  ];

  const baseProps = {
    clearPreviewPlan: jest.fn(),
    flowGoals: "",
    flowInitialState: "{}",
    previewPlan: null,
    setFlowGoals: jest.fn(),
    setFlowInitialState: jest.fn(),
    stepId: "current-step",
    steps,
    updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockApplyFlowGoalSelectionChange.mockResolvedValue(undefined);
    mockUseFlowFormStepFiltering.mockReturnValue({
      included: new Set(),
      satisfied: new Set(),
      missingByStep: new Map(),
    });
  });

  test("renders selectable goal chips except the current step", () => {
    render(<StepEditorFlowConfiguration {...baseProps} />);

    expect(
      screen.getByText(t("stepEditor.flowGoalsLabel"))
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "current-step" })
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "alpha" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "beta" })).toBeInTheDocument();
  });

  test("toggles a goal by delegating selection change", async () => {
    render(<StepEditorFlowConfiguration {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: "alpha" }));

    await waitFor(() => {
      expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
        expect.objectContaining({
          stepIds: ["alpha"],
        })
      );
    });
  });

  test("disables chips already included by the preview plan", () => {
    mockUseFlowFormStepFiltering.mockReturnValue({
      included: new Set(["alpha"]),
      satisfied: new Set(),
      missingByStep: new Map(),
    });

    render(<StepEditorFlowConfiguration {...baseProps} />);

    expect(screen.getByRole("button", { name: "alpha" })).toBeDisabled();
    expect(
      screen
        .getByRole("button", {
          name: "alpha",
        })
        .getAttribute("title")
    ).toBe(t("flowCreate.tooltipAlreadyIncluded"));
  });
});
