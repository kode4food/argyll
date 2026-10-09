import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { AttributeRole, AttributeType, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorFlowConfiguration from "./StepEditorFlowConfiguration";

const mockApplyFlowGoalSelectionChange = jest.fn();
const mockUseFlowFormStepFiltering = jest.fn();
let mockSpaces: { id: string; name: string; selector: object }[] = [];
let mockSpaceSelection: Record<string, Set<string>> = {};

jest.mock("@/app/store/flowStore", () => ({
  useSpaces: () => mockSpaces,
  useSpaceSelection: () => mockSpaceSelection,
}));

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
      flow: { goals: { steps: [] } },
    },
    {
      id: "stock-reservation",
      name: "Stock Reservation",
      type: "service",
      attributes: {
        reservation: { role: AttributeRole.Output, type: AttributeType.String },
      },
    },
    {
      id: "notification-sender",
      name: "Notification Sender",
      type: "service",
      attributes: {
        reservation: {
          role: AttributeRole.Required,
          type: AttributeType.String,
        },
      },
    },
    {
      id: "eligibility-checker",
      name: "Eligibility Checker",
      type: "service",
      attributes: {},
    },
  ];

  const baseProps = {
    clearPreviewPlan: jest.fn(),
    flowCompensate: false,
    flowGoals: [] as string[][],
    flowInitialState: "{}",
    previewPlan: null,
    setFlowCompensate: jest.fn(),
    setFlowGoals: jest.fn(),
    setFlowInitialState: jest.fn(),
    flowSpaceId: "",
    setFlowSpaceId: jest.fn(),
    stepId: "current-step",
    steps,
    updatePreviewPlan: jest.fn().mockResolvedValue(undefined),
  };

  // Opens the suggestion list of one OR line and returns its option names
  const lineSuggestions = (lineIdx: number): string[] => {
    const line = screen.getAllByTestId("flow-goal-line")[lineIdx];
    const trigger = line.querySelector(
      'button[aria-label="Show suggestions"]'
    ) as HTMLButtonElement;
    if (trigger.disabled) return [];
    fireEvent.click(trigger);
    const names = screen
      .getAllByRole("option")
      .map((option) => option.textContent || "");
    fireEvent.click(trigger);
    return names;
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockSpaces = [];
    mockSpaceSelection = {};
    mockApplyFlowGoalSelectionChange.mockResolvedValue(undefined);
    mockUseFlowFormStepFiltering.mockReturnValue({
      satisfied: new Set(),
      blockedByStep: new Map(),
      missingByStep: new Map(),
    });
  });

  test("suggests every Step except the current one", () => {
    render(<StepEditorFlowConfiguration {...baseProps} />);

    expect(
      screen.getByText(t("stepEditor.flowGoalsLabel"))
    ).toBeInTheDocument();
    expect(lineSuggestions(0)).toEqual([
      "eligibility-checker",
      "notification-sender",
      "stock-reservation",
    ]);
  });

  test("adds a goal by delegating selection change", async () => {
    render(<StepEditorFlowConfiguration {...baseProps} />);

    fireEvent.change(screen.getByLabelText(t("goals.addGoal")), {
      target: { value: "eligibility-checker" },
    });

    await waitFor(() => {
      expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
        expect.objectContaining({ goals: [["eligibility-checker"]] })
      );
    });
  });

  test("removes a goal from its line", async () => {
    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["eligibility-checker"], ["stock-reservation"]]}
      />
    );
    mockApplyFlowGoalSelectionChange.mockClear();

    fireEvent.click(
      screen.getByRole("button", {
        name: t("goals.removeGoal", { id: "eligibility-checker" }),
      })
    );

    await waitFor(() => {
      expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
        expect.objectContaining({ goals: [["stock-reservation"]] })
      );
    });
  });

  test("toggles compensate on failure", () => {
    render(<StepEditorFlowConfiguration {...baseProps} />);

    fireEvent.click(screen.getByRole("checkbox"));

    expect(baseProps.setFlowCompensate).toHaveBeenCalledWith(true);
  });

  test("excludes upstream Steps of goals on the same line", () => {
    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["notification-sender"]]}
      />
    );

    expect(lineSuggestions(0)).toEqual(["eligibility-checker"]);
  });

  test("upstream exclusions do not leak across OR lines", () => {
    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["notification-sender"], ["eligibility-checker"]]}
      />
    );

    expect(screen.getByText(t("goals.fallbackTo"))).toBeVisible();
    expect(screen.getByText(t("goals.fallbackTo"))).not.toHaveAttribute(
      "title"
    );
    expect(lineSuggestions(0)).toEqual(["eligibility-checker"]);
    expect(lineSuggestions(1)).toEqual([
      "notification-sender",
      "stock-reservation",
    ]);
  });

  test("prunes an upstream goal when its downstream goal is added", async () => {
    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["eligibility-checker"], ["stock-reservation"]]}
      />
    );
    mockApplyFlowGoalSelectionChange.mockClear();

    fireEvent.change(screen.getAllByLabelText(t("goals.addGoal"))[1], {
      target: { value: "notification-sender" },
    });

    await waitFor(() => {
      expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
        expect.objectContaining({
          goals: [["eligibility-checker"], ["notification-sender"]],
        })
      );
    });
  });

  test("removing a goal set refreshes the plan for the remaining set", () => {
    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["notification-sender"], ["eligibility-checker"]]}
      />
    );

    expect(
      screen.getAllByTestId("flow-goal-line")[0].parentElement
    ).toContainElement(screen.getByText(t("goals.fallbackTo")));
    mockApplyFlowGoalSelectionChange.mockClear();
    fireEvent.click(
      screen.getAllByRole("button", { name: t("goals.removeSet") })[0]
    );
    expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
      expect.objectContaining({
        goals: [["eligibility-checker"]],
        setInitialState: baseProps.setFlowInitialState,
        updatePreviewPlan: baseProps.updatePreviewPlan,
      })
    );
  });

  test("OR adds a line only when the last line has goals", () => {
    const { rerender } = render(<StepEditorFlowConfiguration {...baseProps} />);
    const orButton = () =>
      screen.getByRole("button", { name: t("goals.addFallback") });
    expect(orButton()).toBeDisabled();

    rerender(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["eligibility-checker"]]}
      />
    );
    fireEvent.click(orButton());
    expect(baseProps.setFlowGoals).toHaveBeenCalledWith([
      ["eligibility-checker"],
      [],
    ]);
  });

  test("prunes goals immediately when Space changes", async () => {
    mockSpaces = [
      {
        id: "alpha-space",
        name: "Alpha Space",
        selector: { language: "lua", script: "return true" },
      },
    ];
    mockSpaceSelection = {
      "alpha-space": new Set(["eligibility-checker"]),
    };

    render(
      <StepEditorFlowConfiguration
        {...baseProps}
        flowGoals={[["eligibility-checker", "stock-reservation"]]}
      />
    );

    await waitFor(() => {
      expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalled();
    });
    jest.clearAllMocks();

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.spaceLabel") })
    );
    fireEvent.click(screen.getByRole("option", { name: "Alpha Space" }));

    expect(baseProps.setFlowSpaceId).toHaveBeenCalledWith("alpha-space");
    expect(baseProps.setFlowGoals).toHaveBeenCalledWith([
      ["eligibility-checker"],
    ]);
    expect(baseProps.clearPreviewPlan).toHaveBeenCalled();
    expect(mockApplyFlowGoalSelectionChange).toHaveBeenCalledWith(
      expect.objectContaining({
        goals: [["eligibility-checker"]],
        spaceId: "alpha-space",
      })
    );
  });
});
