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

  test("prunes upstream goals when included by last goal", async () => {
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

    const combinedPlan: ExecutionPlan = {
      goals: ["order-creator", "notification-sender"],
      required: [],
      steps: {
        "order-creator": orderCreator,
        "notification-sender": notificationSender,
      },
      attributes: {},
    };

    const lastGoalPlan: ExecutionPlan = {
      goals: ["notification-sender"],
      required: [],
      steps: {
        "order-creator": orderCreator,
        "notification-sender": notificationSender,
      },
      attributes: {},
    };

    getExecutionPlan.mockImplementation(async (stepIds: string[]) => {
      if (stepIds.length === 1 && stepIds[0] === "notification-sender") {
        return lastGoalPlan;
      }
      return combinedPlan;
    });

    const setInitialState = jest.fn();
    const setGoalSteps = jest.fn();
    const setPreviewPlan = jest.fn();
    const updatePreviewPlan = jest.fn().mockResolvedValue(undefined);
    const clearPreviewPlan = jest.fn();
    const setNewID = jest.fn();

    await applyFlowGoalSelectionChange({
      stepIds: ["order-creator", "notification-sender"],
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

    expect(setGoalSteps).toHaveBeenCalledWith(["notification-sender"]);
    expect(setPreviewPlan).toHaveBeenCalledWith(combinedPlan);
    expect(updatePreviewPlan).toHaveBeenCalledWith(["notification-sender"], {});
    expect(clearPreviewPlan).not.toHaveBeenCalled();
    expect(setNewID).toHaveBeenCalledWith("notification-sender-0001");
  });

  test("clears selection and preview when stepIds is empty", async () => {
    const setInitialState = jest.fn();
    const setGoalSteps = jest.fn();
    const setPreviewPlan = jest.fn();
    const updatePreviewPlan = jest.fn().mockResolvedValue(undefined);
    const clearPreviewPlan = jest.fn();

    await applyFlowGoalSelectionChange({
      stepIds: [],
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
    expect(setGoalSteps).toHaveBeenCalledWith([]);
    expect(updatePreviewPlan).not.toHaveBeenCalled();
  });
});
