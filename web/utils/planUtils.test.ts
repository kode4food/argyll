import { getStepsFromPlan } from "./planUtils";
import { ExecutionPlan, Step, AttributeRole, AttributeType } from "../app/api";

describe("planUtils", () => {
  describe("getStepsFromPlan", () => {
    const mockStep: Step = {
      id: "step-1",
      name: "Test Step",
      type: "sync",
      version: "1.0.0",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
      },
      http: {
        endpoint: "http://localhost:8080/test",
        timeout: 5000,
      },
    };

    test("returns empty array when plan is undefined", () => {
      expect(getStepsFromPlan(undefined)).toEqual([]);
    });

    test("returns empty array when plan is null", () => {
      expect(getStepsFromPlan(null)).toEqual([]);
    });

    test("returns empty array when plan has no steps", () => {
      const plan: ExecutionPlan = {
        steps: {},
        attributes: {},
        goals: [],
        required: [],
      };
      expect(getStepsFromPlan(plan)).toEqual([]);
    });

    test("returns steps from plan", () => {
      const plan: ExecutionPlan = {
        steps: {
          "step-1": mockStep,
          "step-2": { ...mockStep, id: "step-2", name: "Step 2" },
        },
        attributes: {},
        goals: [],
        required: [],
      };

      const result = getStepsFromPlan(plan);

      expect(result).toHaveLength(2);
      expect(result[0].id).toBe("step-1");
      expect(result[1].id).toBe("step-2");
    });

    test("returns steps directly without wrapper", () => {
      const plan: ExecutionPlan = {
        steps: {
          "step-1": mockStep,
        },
        attributes: {},
        goals: [],
        required: [],
      };

      const result = getStepsFromPlan(plan);

      expect(result[0]).toEqual(mockStep);
    });
  });
});
