import { fireEvent, render, screen } from "@testing-library/react";
import { AttributeRole, AttributeType, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import FlowGoalsSection from "./FlowGoalsSection";

describe("FlowGoalsSection", () => {
  const steps: Step[] = [
    {
      id: "step-1",
      name: "Step One",
      type: "sync",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
      },
      http: { endpoint: "http://localhost:8080/test", timeout: 5000 },
    },
  ];

  test("renders step count and create action", () => {
    const onCreateStep = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map()}
        included={new Set()}
        missingByStep={new Map()}
        onCreateStep={onCreateStep}
        onGoalStepsChange={jest.fn()}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
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

  test("toggles a goal step when clicked", () => {
    const onGoalStepsChange = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map()}
        included={new Set()}
        missingByStep={new Map()}
        onGoalStepsChange={onGoalStepsChange}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
        stepsCount={1}
      />
    );

    fireEvent.click(screen.getByText("Step One"));
    expect(onGoalStepsChange).toHaveBeenCalledWith(["step-1"]);
  });

  test("disables a step blocked by initial state", () => {
    const onGoalStepsChange = jest.fn();

    render(
      <FlowGoalsSection
        goalSteps={[]}
        blockedByStep={new Map([["step-1", ["input1"]]])}
        included={new Set()}
        missingByStep={new Map()}
        onGoalStepsChange={onGoalStepsChange}
        satisfied={new Set()}
        showBottomFade={false}
        showTopFade={false}
        sidebarListRef={{ current: null }}
        sortedSteps={steps}
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
});
