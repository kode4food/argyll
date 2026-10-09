import { applyFlowGoalSelectionChange } from "./flowGoalSelectionModel";
import { api, ExecutionPlan, Step } from "@/app/api";

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: { getExecutionPlan: jest.fn() },
}));

jest.mock("@/utils/flowUtils", () => ({
  ...jest.requireActual("@/utils/flowUtils"),
  generatePadded: () => "0001",
}));

const getExecutionPlan = api.getExecutionPlan as jest.Mock;

describe("flowGoalSelectionModel", () => {
  beforeEach(() => {
    getExecutionPlan.mockReset();
  });

  test("plans every non-empty goal set", async () => {
    const orderCreator: Step = {
      id: "order-creator",
      name: "Order Creator",
      type: "service",
      attributes: {},
    };
    const notificationSender: Step = {
      id: "notification-sender",
      name: "Notification Sender",
      type: "service",
      attributes: {},
    };

    const plan: ExecutionPlan = {
      goals: {
        steps: ["order-creator"],
        else: { steps: ["notification-sender"] },
      },
      required: [],
      steps: {
        "order-creator": orderCreator,
        "notification-sender": notificationSender,
      },
      attributes: {},
    };
    getExecutionPlan.mockResolvedValue(plan);

    const setInitialState = jest.fn();
    const setGoalSteps = jest.fn();
    const setPreviewPlan = jest.fn();
    const updatePreviewPlan = jest.fn().mockResolvedValue(undefined);
    const clearPreviewPlan = jest.fn();
    const setNewID = jest.fn();
    const goals = [["order-creator"], ["notification-sender"], []];

    await applyFlowGoalSelectionChange({
      goals,
      initialState: "{}",
      steps: [orderCreator, notificationSender],
      idManuallyEdited: false,
      setNewID,
      setInitialState,
      setGoalSteps,
      setPreviewPlan,
      updatePreviewPlan,
      clearPreviewPlan,
    });

    expect(getExecutionPlan).toHaveBeenCalledWith({
      goalSteps: [["order-creator"], ["notification-sender"]],
      initialState: {},
      spaceId: undefined,
    });
    expect(setGoalSteps).toHaveBeenCalledWith(goals);
    expect(setPreviewPlan).toHaveBeenCalledWith(plan);
    expect(updatePreviewPlan).toHaveBeenCalledWith(
      [["order-creator"], ["notification-sender"]],
      {},
      undefined
    );
    expect(clearPreviewPlan).not.toHaveBeenCalled();
    expect(setNewID).toHaveBeenCalledWith("notification-sender-0001");
  });

  test("clears preview when every goal set is empty", async () => {
    const setInitialState = jest.fn();
    const setGoalSteps = jest.fn();
    const setPreviewPlan = jest.fn();
    const updatePreviewPlan = jest.fn().mockResolvedValue(undefined);
    const clearPreviewPlan = jest.fn();

    await applyFlowGoalSelectionChange({
      goals: [[]],
      initialState: "{}",
      steps: [],
      setInitialState,
      setGoalSteps,
      setPreviewPlan,
      updatePreviewPlan,
      clearPreviewPlan,
    });

    expect(clearPreviewPlan).toHaveBeenCalled();
    expect(setPreviewPlan).toHaveBeenCalledWith(null);
    expect(setGoalSteps).toHaveBeenCalledWith([[]]);
    expect(updatePreviewPlan).not.toHaveBeenCalled();
  });
});
