import { AttributeRole, AttributeType, ExecutionPlan } from "@/app/api";
import { getFlowPlanAttributeOptions } from "./flowPlanAttributeOptions";

describe("flowPlanAttributeOptions", () => {
  it("returns empty options when plan is null", () => {
    expect(getFlowPlanAttributeOptions(null)).toEqual({
      flowInputOptions: [],
      flowOutputOptions: [],
    });
  });

  it("marks required only when input is externally required by the plan", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-step"],
      required: ["order_id"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync",
          attributes: {
            order_id: {
              role: AttributeRole.Required,
              type: AttributeType.String,
            },
            quantity: {
              role: AttributeRole.Required,
              type: AttributeType.Number,
            },
            notes: {
              role: AttributeRole.Optional,
              type: AttributeType.String,
            },
            total_price: {
              role: AttributeRole.Output,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        upstream: {
          id: "upstream",
          name: "Upstream",
          type: "sync",
          attributes: {
            quantity: {
              role: AttributeRole.Output,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/up", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan)).toEqual({
      flowInputOptions: [
        { name: "notes", required: false, type: AttributeType.String },
        { name: "order_id", required: true, type: AttributeType.String },
        {
          name: "quantity",
          required: false,
          type: AttributeType.Number,
          defaultValue: "0",
        },
      ],
      flowOutputOptions: ["quantity", "total_price"],
    });
  });

  it("keeps required true when the same input appears in multiple steps", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-a"],
      required: ["user_id"],
      attributes: {},
      steps: {
        "goal-a": {
          id: "goal-a",
          name: "Goal A",
          type: "sync",
          attributes: {
            user_id: {
              role: AttributeRole.Required,
              type: AttributeType.String,
            },
          },
          http: { endpoint: "http://localhost:8080/a", timeout: 5000 },
        },
        "goal-b": {
          id: "goal-b",
          name: "Goal B",
          type: "sync",
          attributes: {
            user_id: {
              role: AttributeRole.Optional,
              type: AttributeType.String,
            },
          },
          http: { endpoint: "http://localhost:8080/b", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      { name: "user_id", required: true, type: AttributeType.String },
    ]);
  });

  it("normalizes and carries default values for flow inputs", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-a"],
      required: ["required_with_default"],
      attributes: {},
      steps: {
        "goal-a": {
          id: "goal-a",
          name: "Goal A",
          type: "sync",
          attributes: {
            required_with_default: {
              role: AttributeRole.Required,
              type: AttributeType.String,
              default: '"same-value"',
            },
            optional_with_default: {
              role: AttributeRole.Optional,
              type: AttributeType.Number,
              default: "42",
            },
            optional_without_default: {
              role: AttributeRole.Optional,
              type: AttributeType.String,
            },
          },
          http: { endpoint: "http://localhost:8080/a", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      {
        name: "optional_with_default",
        required: false,
        type: AttributeType.Number,
        defaultValue: "42",
      },
      {
        name: "optional_without_default",
        required: false,
        type: AttributeType.String,
      },
      {
        name: "required_with_default",
        required: true,
        type: AttributeType.String,
        defaultValue: "same-value",
      },
    ]);
  });
});
