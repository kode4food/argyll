import { AttributeRole, ExecutionPlan, Step } from "@/app/api";
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
      goals: { steps: ["goal-step"] },
      required: ["order_id"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            order_id: {
              role: AttributeRole.Required,
            },
            quantity: {
              role: AttributeRole.Required,
            },
            notes: {
              role: AttributeRole.Optional,
            },
            total_price: {
              role: AttributeRole.Output,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        upstream: {
          id: "upstream",
          name: "Upstream",
          type: "service",
          attributes: {
            quantity: {
              role: AttributeRole.Output,
            },
          },
          http: { endpoint: "http://localhost:8080/up", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan)).toEqual({
      flowInputOptions: [
        { name: "order_id", required: true },
        { name: "notes", required: false },
        {
          name: "quantity",
          required: false,
          satisfiedByOutput: true,
        },
      ],
      flowOutputOptions: ["quantity", "total_price"],
    });
  });

  it("keeps required true when the same input appears in multiple steps", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-a"] },
      required: ["user_id"],
      attributes: {},
      steps: {
        "goal-a": {
          id: "goal-a",
          name: "Goal A",
          type: "service",
          attributes: {
            user_id: {
              role: AttributeRole.Required,
            },
          },
          http: { endpoint: "http://localhost:8080/a", timeout: 5000 },
        },
        "goal-b": {
          id: "goal-b",
          name: "Goal B",
          type: "service",
          attributes: {
            user_id: {
              role: AttributeRole.Optional,
            },
          },
          http: { endpoint: "http://localhost:8080/b", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      { name: "user_id", required: true },
    ]);
  });

  it("normalizes and carries default values for flow inputs", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-a"] },
      required: ["required_with_default"],
      attributes: {},
      steps: {
        "goal-a": {
          id: "goal-a",
          name: "Goal A",
          type: "service",
          attributes: {
            required_with_default: {
              role: AttributeRole.Required,
            },
            optional_with_default: {
              role: AttributeRole.Optional,
              optional: { default: "42" },
            },
            optional_without_default: {
              role: AttributeRole.Optional,
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
      },
      {
        name: "optional_without_default",
        required: false,
      },
      {
        name: "optional_with_default",
        required: false,
        defaultValue: "42",
      },
    ]);
  });

  it("places upstream-fulfilled inputs after local optional groups", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-step"] },
      required: ["order_id"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            order_id: {
              role: AttributeRole.Required,
            },
            notes: {
              role: AttributeRole.Optional,
            },
            quantity: {
              role: AttributeRole.Required,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        upstream: {
          id: "upstream",
          name: "Upstream",
          type: "service",
          attributes: {
            quantity: {
              role: AttributeRole.Output,
            },
          },
          http: { endpoint: "http://localhost:8080/up", timeout: 5000 },
        },
      },
    };

    expect(getFlowPlanAttributeOptions(plan).flowInputOptions).toEqual([
      { name: "order_id", required: true },
      { name: "notes", required: false },
      {
        name: "quantity",
        required: false,
        satisfiedByOutput: true,
      },
    ]);
  });

  it("sorts optional inputs without a spec default first", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-step"] },
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            alpha_number: {
              role: AttributeRole.Optional,
            },
            beta_with_default: {
              role: AttributeRole.Optional,
              optional: { default: '"fallback"' },
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
      },
      {
        name: "beta_with_default",
        required: false,
        defaultValue: "fallback",
      },
    ]);
  });

  it("prefers explicit spec defaults over type-derived defaults", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["price-calculator", "order-creator"] },
      required: ["quantity"],
      attributes: {},
      steps: {
        "price-calculator": {
          id: "price-calculator",
          name: "Price Calculator",
          type: "service",
          attributes: {
            quantity: {
              role: AttributeRole.Required,
            },
          },
          http: { endpoint: "http://localhost:8080/price", timeout: 5000 },
        },
        "order-creator": {
          id: "order-creator",
          name: "Order Creator",
          type: "service",
          attributes: {
            quantity: {
              role: AttributeRole.Optional,
              optional: { default: "1" },
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
        defaultValue: "1",
      },
    ]);
  });

  it("uses required as merged status when any step requires the input", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-step"] },
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            alpha: { role: AttributeRole.Optional },
            shared: {
              role: AttributeRole.Optional,
            },
          },
          http: { endpoint: "http://localhost:8080/goal", timeout: 5000 },
        },
        validator: {
          id: "validator",
          name: "Validator",
          type: "service",
          attributes: {
            shared: {
              role: AttributeRole.Required,
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
      },
      { name: "alpha", required: false },
    ]);
  });

  it("includes unreachable attributes from excluded.missing", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-step"] },
      required: ["order_id"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            order_id: {
              role: AttributeRole.Required,
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
        type: "service",
        attributes: {
          order_id: {
            role: AttributeRole.Required,
          },
        },
      },
      {
        id: "excluded-step",
        name: "Excluded Step",
        type: "service",
        attributes: {
          age: {
            role: AttributeRole.Required,
          },
          income: {
            role: AttributeRole.Required,
          },
          result: {
            role: AttributeRole.Output,
          },
        },
      },
    ];

    const result = getFlowPlanAttributeOptions(plan, catalogSteps);
    expect(result.flowInputOptions).toEqual([
      { name: "order_id", required: true },
      {
        name: "age",
        required: false,
        unreachable: true,
      },
      {
        name: "income",
        required: false,
        unreachable: true,
      },
    ]);
  });

  it("skips unreachable attributes already available in plan", () => {
    const plan: ExecutionPlan = {
      goals: { steps: ["goal-step"] },
      required: ["shared"],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
          attributes: {
            shared: {
              role: AttributeRole.Required,
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
        type: "service",
        attributes: {
          shared: {
            role: AttributeRole.Required,
          },
        },
      },
      {
        id: "excluded-step",
        name: "Excluded Step",
        type: "service",
        attributes: {
          shared: {
            role: AttributeRole.Required,
          },
          extra: {
            role: AttributeRole.Required,
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
      goals: { steps: ["goal-step"] },
      required: [],
      attributes: {},
      steps: {
        "goal-step": {
          id: "goal-step",
          name: "Goal Step",
          type: "service",
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
