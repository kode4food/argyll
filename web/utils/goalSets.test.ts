import { AttributeRole, AttributeType, Step } from "@/app/api";
import {
  activeGoalSet,
  addGoal,
  formatGoals,
  nonEmptyGoalSets,
  toggleGoal,
  upstreamSteps,
  withActiveGoalSet,
} from "./goalSets";

const STOCK_RESERVATION = "stock-reservation";
const NOTIFICATION_SENDER = "notification-sender";
const ELIGIBILITY_CHECKER = "eligibility-checker";

export const goalSetSteps: Step[] = [
  {
    id: STOCK_RESERVATION,
    name: "Stock Reservation",
    type: "service",
    attributes: {
      reservation: { role: AttributeRole.Output, type: AttributeType.String },
    },
  },
  {
    id: NOTIFICATION_SENDER,
    name: "Notification Sender",
    type: "service",
    attributes: {
      reservation: {
        role: AttributeRole.Required,
        type: AttributeType.String,
      },
      notified: { role: AttributeRole.Output, type: AttributeType.Boolean },
    },
  },
  {
    id: ELIGIBILITY_CHECKER,
    name: "Eligibility Checker",
    type: "service",
    attributes: {
      eligible: { role: AttributeRole.Output, type: AttributeType.Boolean },
    },
  },
];

describe("goalSets", () => {
  test("upstream exclusions stay within one AND-set", () => {
    expect(
      upstreamSteps(goalSetSteps, [NOTIFICATION_SENDER]).has(STOCK_RESERVATION)
    ).toBe(true);
    expect(
      upstreamSteps(goalSetSteps, [ELIGIBILITY_CHECKER]).has(STOCK_RESERVATION)
    ).toBe(false);
  });

  test("adding a downstream goal prunes its upstream goal", () => {
    expect(
      addGoal(goalSetSteps, [STOCK_RESERVATION], NOTIFICATION_SENDER)
    ).toEqual([NOTIFICATION_SENDER]);
    expect(
      addGoal(goalSetSteps, [ELIGIBILITY_CHECKER], STOCK_RESERVATION)
    ).toEqual([ELIGIBILITY_CHECKER, STOCK_RESERVATION]);
  });

  test("toggles goals in a set", () => {
    expect(toggleGoal(goalSetSteps, ["a", "b"], "a")).toEqual(["b"]);
    expect(toggleGoal(goalSetSteps, ["a"], "b")).toEqual(["a", "b"]);
  });

  test("edits only the last set", () => {
    expect(activeGoalSet([])).toEqual([]);
    expect(activeGoalSet([["a"], ["b"]])).toEqual(["b"]);
    expect(withActiveGoalSet([], ["a"])).toEqual([["a"]]);
    expect(withActiveGoalSet([["a"], []], ["b"])).toEqual([["a"], ["b"]]);
    expect(nonEmptyGoalSets([["a"], []])).toEqual([["a"]]);
    expect(formatGoals([["a", "b"], ["c"]])).toBe("a, b ELSE c");
  });
});
