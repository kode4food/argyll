import { fireEvent, render, screen } from "@testing-library/react";
import { AttributeRole, AttributeType, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import FlowGoalsSection from "./FlowGoalsSection";

describe("FlowGoalsSection", () => {
  const steps: Step[] = [
    {
      id: "step-1",
      name: "Step One",
      type: "service",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
      },
      http: { endpoint: "http://localhost:8080/test", timeout: 5000 },
    },
    {
      id: "step-2",
      name: "Step Two",
      type: "service",
      attributes: {},
      http: { endpoint: "http://localhost:8080/test", timeout: 5000 },
    },
  ];

  test("renders step count and create action", () => {
    const onCreateStep = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map()}
        missingByStep={new Map()}
        onCreateStep={onCreateStep}
        onGoalStepsChange={jest.fn()}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        spaceScoped={false}
        stepsCount={1}
      />
    );

    expect(
      screen.getByText(t("overview.stepsRegistered", { count: 1 }))
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("overview.addStep") })
    );
    expect(onCreateStep).toHaveBeenCalled();
  });

  test("seeds from goal steps when the create action is shift-clicked", () => {
    const onCreateStep = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[["step-1"]]}
        blockedByStep={new Map()}
        missingByStep={new Map()}
        onCreateStep={onCreateStep}
        onGoalStepsChange={jest.fn()}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        spaceScoped={false}
        stepsCount={2}
      />
    );

    const button = screen.getByRole("button", { name: t("overview.addStep") });

    fireEvent.click(button);
    expect(onCreateStep).toHaveBeenLastCalledWith(false);

    fireEvent.keyDown(window, { key: "Shift", shiftKey: true });
    expect(
      screen.getByRole("button", {
        name: t("overview.addFlowStepFromGoals"),
      })
    ).toBeInTheDocument();

    fireEvent.click(button, { shiftKey: true });
    expect(onCreateStep).toHaveBeenLastCalledWith(true);
  });

  test("toggles a goal step when clicked", () => {
    const onGoalStepsChange = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map()}
        missingByStep={new Map()}
        onGoalStepsChange={onGoalStepsChange}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        spaceScoped={false}
        stepsCount={1}
      />
    );

    fireEvent.click(screen.getByText("Step One"));
    expect(onGoalStepsChange).toHaveBeenCalledWith([["step-1"]]);
  });

  test("disables a step blocked by initial state", () => {
    const onGoalStepsChange = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map([["step-1", ["input1"]]])}
        missingByStep={new Map()}
        onGoalStepsChange={onGoalStepsChange}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        spaceScoped={false}
        stepsCount={1}
      />
    );

    const item = screen
      .getByText("Step One")
      .closest('div[title="Blocked by initial state: input1"]');
    expect(item).toBeInTheDocument();

    fireEvent.click(screen.getByText("Step One"));
    expect(onGoalStepsChange).not.toHaveBeenCalled();
  });

  test("moves focus between goal steps with arrow keys", () => {
    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map()}
        missingByStep={new Map()}
        onGoalStepsChange={jest.fn()}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        spaceScoped={false}
        stepsCount={2}
      />
    );

    const first = screen.getByRole("button", { name: /Step One/ });
    const second = screen.getByRole("button", { name: /Step Two/ });

    first.focus();
    fireEvent.keyDown(first, { key: "ArrowDown" });
    expect(second).toHaveFocus();

    fireEvent.keyDown(second, { key: "ArrowUp" });
    expect(first).toHaveFocus();
  });

  describe("fallback goal sets", () => {
    const catalog: Step[] = [
      {
        id: "stock-reservation",
        name: "Stock Reservation",
        type: "service",
        attributes: {
          reservation: {
            role: AttributeRole.Output,
            type: AttributeType.String,
          },
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

    const renderGoals = (
      goalSteps: string[][],
      onGoalStepsChange = jest.fn()
    ) =>
      render(
        <FlowGoalsSection
          goalSteps={goalSteps}
          blockedByStep={new Map()}
          missingByStep={new Map()}
          onGoalStepsChange={onGoalStepsChange}
          satisfied={new Set()}
          showBottomFade={false}
          showTopFade={false}
          sidebarListRef={{ current: null }}
          sortedSteps={catalog}
          spaceScoped={false}
          stepsCount={catalog.length}
        />
      );

    const stockItem = () =>
      screen.getByText("Stock Reservation").closest('[role="button"]');

    test("disables upstream Steps within the same AND-set", () => {
      const onGoalStepsChange = jest.fn();
      renderGoals([["notification-sender"]], onGoalStepsChange);

      expect(stockItem()).toHaveAttribute("aria-disabled", "true");
      fireEvent.click(screen.getByText("Stock Reservation"));
      expect(onGoalStepsChange).not.toHaveBeenCalled();
    });

    test("prunes an upstream goal when its downstream goal is added", () => {
      const onGoalStepsChange = jest.fn();
      renderGoals([["stock-reservation"]], onGoalStepsChange);

      fireEvent.click(screen.getByText("Notification Sender"));
      expect(onGoalStepsChange).toHaveBeenCalledWith([["notification-sender"]]);
    });

    test("does not leak upstream exclusions across OR sets", () => {
      const onGoalStepsChange = jest.fn();
      renderGoals(
        [["notification-sender"], ["eligibility-checker"]],
        onGoalStepsChange
      );

      expect(screen.getByTestId("goal-set-summary")).toHaveTextContent(
        "notification-sender"
      );
      expect(
        screen.getByTestId("goal-set-summary").parentElement
      ).toContainElement(screen.getByText(t("goals.fallbackTo")));
      expect(stockItem()).toHaveAttribute("aria-disabled", "false");
      fireEvent.click(screen.getByText("Stock Reservation"));
      expect(onGoalStepsChange).toHaveBeenCalledWith([
        ["notification-sender"],
        ["eligibility-checker", "stock-reservation"],
      ]);
    });

    test("removes a frozen goal set", () => {
      const onGoalStepsChange = jest.fn();
      renderGoals(
        [["notification-sender"], ["stock-reservation"], []],
        onGoalStepsChange
      );

      fireEvent.click(
        screen.getAllByRole("button", { name: t("goals.removeSet") })[0]
      );
      expect(onGoalStepsChange).toHaveBeenCalledWith([
        ["stock-reservation"],
        [],
      ]);
    });

    test("OR starts a new set only when the active set has goals", () => {
      const onGoalStepsChange = jest.fn();
      const { rerender } = renderGoals([[]], onGoalStepsChange);
      const orButton = () =>
        screen.getByRole("button", { name: t("goals.addFallback") });
      expect(orButton()).toBeDisabled();

      rerender(
        <FlowGoalsSection
          goalSteps={[["notification-sender"]]}
          blockedByStep={new Map()}
          missingByStep={new Map()}
          onGoalStepsChange={onGoalStepsChange}
          satisfied={new Set()}
          showBottomFade={false}
          showTopFade={false}
          sidebarListRef={{ current: null }}
          sortedSteps={catalog}
          spaceScoped={false}
          stepsCount={catalog.length}
        />
      );
      fireEvent.click(orButton());
      expect(onGoalStepsChange).toHaveBeenCalledWith([
        ["notification-sender"],
        [],
      ]);
    });
  });
});
