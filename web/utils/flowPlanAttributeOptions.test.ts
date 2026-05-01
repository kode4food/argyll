import { AttributeRole, AttributeType, ExecutionPlan, Step } from "@/app/api";
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
        { name: "order_id", required: true, type: AttributeType.String },
        { name: "notes", required: false, type: AttributeType.String },
        {
          name: "quantity",
          required: false,
          type: AttributeType.Number,
          defaultValue: "0",
          satisfiedByOutput: true,
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
              input: { default: '"same-value"' },
            },
            optional_with_default: {
              role: AttributeRole.Optional,
              type: AttributeType.Number,
              input: { default: "42" },
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
        name: "required_with_default",
        required: true,
        type: AttributeType.String,
        defaultValue: "same-value",
      },
      {
        name: "optional_without_default",
        required: false,
        type: AttributeType.String,
      },
      {
        name: "optional_with_default",
        required: false,
        type: AttributeType.Number,
        defaultValue: "42",
      },
    ]);
  });

  it("places upstream-fulfilled inputs after local optional groups", () => {
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
            notes: {
              role: AttributeRole.Optional,
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

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      { name: "order_id", required: true, type: AttributeType.String },
      { name: "notes", required: false, type: AttributeType.String },
      {
        name: "quantity",
        required: false,
        type: AttributeType.Number,
        defaultValue: "0",
        satisfiedByOutput: true,
      },
    ]);
  });

  it("treats type-based defaults as optional-without-spec-default for sorting", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-step"],
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync",
          attributes: {
            alpha_number: {
              role: AttributeRole.Optional,
              type: AttributeType.Number,
            },
            beta_with_default: {
              role: AttributeRole.Optional,
              type: AttributeType.String,
              input: { default: '"fallback"' },
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      {
        name: "alpha_number",
        required: false,
        type: AttributeType.Number,
        defaultValue: "0",
      },
      {
        name: "beta_with_default",
        required: false,
        type: AttributeType.String,
        defaultValue: "fallback",
      },
    ]);
  });

  it("prefers explicit spec defaults over type-derived defaults", () => {
    const plan: ExecutionPlan = {
      goals: ["price-calculator", "order-creator"],
      required: ["quantity"],
      attributes: {},
      steps: {
        "price-calculator": {
          id: "price-calculator",
          name: "Price Calculator",
          type: "sync",
          attributes: {
            quantity: {
              role: AttributeRole.Required,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/price", timeout: 5000 },
        },
        "order-creator": {
          id: "order-creator",
          name: "Order Creator",
          type: "sync",
          attributes: {
            quantity: {
              role: AttributeRole.Optional,
              type: AttributeType.Number,
              input: { default: "1" },
            },
          },
          http: { endpoint: "http://localhost:8080/order", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      {
        name: "quantity",
        required: true,
        type: AttributeType.Number,
        defaultValue: "1",
      },
    ]);
  });

  it("uses required as merged status when any step requires the input", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-step"],
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync",
          attributes: {
            alpha: { role: AttributeRole.Optional, type: AttributeType.String },
            shared: {
              role: AttributeRole.Optional,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        validator: {
          id: "validator",
          name: "Validator",
          type: "sync",
          attributes: {
            shared: {
              role: AttributeRole.Required,
              type: AttributeType.Number,
            },
          },
          http: { endpoint: "http://localhost:8080/validator", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      {
        name: "shared",
        required: true,
        type: AttributeType.Number,
        defaultValue: "0",
      },
      { name: "alpha", required: false, type: AttributeType.String },
    ]);
  });

  it("includes unreachable attributes from excluded.missing", () => {
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
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
      },
      excluded: {
        missing: {
          "excluded-step": ["age", "income"],
        },
      },
    };

    const catalogSteps: Step[] = [
      {
        id: "goal-step",
        name: "Goal Step",
        type: "sync",
        attributes: {
          order_id: {
            role: AttributeRole.Required,
            type: AttributeType.String,
          },
        },
      },
      {
        id: "excluded-step",
        name: "Excluded Step",
        type: "sync",
        attributes: {
          age: {
            role: AttributeRole.Required,
            type: AttributeType.Number,
          },
          income: {
            role: AttributeRole.Required,
            type: AttributeType.Number,
          },
          result: {
            role: AttributeRole.Output,
            type: AttributeType.Object,
          },
        },
      },
    ];

    const result = getFlowPlanAttributeOptions(plan, catalogSteps);
    expect(result.flowInputOptions).toEqual([
      { name: "order_id", required: true, type: AttributeType.String },
      {
        name: "age",
        required: false,
        type: AttributeType.Number,
        unreachable: true,
      },
      {
        name: "income",
        required: false,
        type: AttributeType.Number,
        unreachable: true,
      },
    ]);
  });

  it("skips unreachable attributes already available in plan", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-step"],
      required: ["shared"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync",
          attributes: {
            shared: {
              role: AttributeRole.Required,
              type: AttributeType.String,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
      },
      excluded: {
        missing: {
          "excluded-step": ["shared", "extra"],
        },
      },
    };

    const catalogSteps: Step[] = [
      {
        id: "goal-step",
        name: "Goal Step",
        type: "sync",
        attributes: {
          shared: {
            role: AttributeRole.Required,
            type: AttributeType.String,
          },
        },
      },
      {
        id: "excluded-step",
        name: "Excluded Step",
        type: "sync",
        attributes: {
          shared: {
            role: AttributeRole.Required,
            type: AttributeType.String,
          },
          extra: {
            role: AttributeRole.Required,
            type: AttributeType.Object,
          },
        },
      },
    ];

    const result = getFlowPlanAttributeOptions(plan, catalogSteps);
    const names = result.flowInputOptions.map((o) => o.name);
    expect(names.filter((n) => n === "shared")).toHaveLength(1);
    expect(names).toContain("extra");
  });

  it("returns no unreachable attributes without catalog steps", () => {
    const plan: ExecutionPlan = {
      goals: ["goal-step"],
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "sync",
          attributes: {},
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
      },
      excluded: {
        missing: { "excluded-step": ["age"] },
      },
    };

    const result = getFlowPlanAttributeOptions(plan);
    expect(result.flowInputOptions).toEqual([]);
  });
});
