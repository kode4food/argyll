import {
  Attribute,
  buildAttributesFromStep,
  createStepAttributes,
  getValidationError,
  validateAttributesList,
} from "./stepEditorUtils";
import { AttributeRole, AttributeType, Step } from "@/app/api";

const attribute = (overrides: Partial<Attribute> = {}): Attribute => ({
  id: "attr-1",
  role: "required",
  name: "param",
  dataType: AttributeType.String,
  ...overrides,
});

describe("stepEditorUtils", () => {
  test("builds editor attributes", () => {
    const step: Step = {
      id: "test-step",
      name: "Test",
      type: "flow",
      attributes: {
        required_arg: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          required: {
            collect: "all",
            for_each: true,
            match: { language: "jpath", script: "$.kind" },
            mapping: {
              name: "child_in",
              script: { language: "lua", script: "return value" },
            },
          },
        },
        const_arg: {
          role: AttributeRole.Const,
          type: AttributeType.String,
          const: { value: '"fixed"' },
        },
        optional_arg: {
          role: AttributeRole.Optional,
          type: AttributeType.Number,
          optional: { collect: "some", default: "42", deadline: 3000 },
        },
        meta_arg: {
          role: AttributeRole.Meta,
          type: AttributeType.String,
          meta: { key: "request_id" },
        },
        output_arg: {
          role: AttributeRole.Output,
          type: AttributeType.String,
          output: { mapping: { name: "child_out" } },
        },
      },
      flow: { goals: ["goal-1"] },
    };

    const result = Object.fromEntries(
      buildAttributesFromStep(step).map((item) => [item.name, item])
    );

    expect(result.required_arg).toEqual(
      expect.objectContaining({
        role: "required",
        collect: "all",
        forEach: true,
        matchLanguage: "jpath",
        matchScript: "$.kind",
        mappingName: "child_in",
        mappingLanguage: "lua",
        mappingScript: "return value",
      })
    );
    expect(result.const_arg.defaultValue).toBe('"fixed"');
    expect(result.optional_arg).toEqual(
      expect.objectContaining({
        role: "optional",
        collect: "some",
        defaultValue: "42",
        deadline: 3000,
      })
    );
    expect(result.meta_arg.metaKey).toBe("request_id");
    expect(result.output_arg.mappingName).toBe("child_out");
    expect(buildAttributesFromStep(null)).toEqual([]);
  });

  const attributeValidationCases: Array<
    [string, Attribute[], ReturnType<typeof validateAttributesList>]
  > = [
    [
      "valid attributes",
      [
        attribute(),
        attribute({ id: "attr-2", role: "output", name: "result" }),
      ],
      null,
    ],
    [
      "empty names",
      [attribute({ name: "   " })],
      { key: "stepEditor.attributeNameRequired" },
    ],
    [
      "duplicate names",
      [attribute(), attribute({ id: "attr-2", role: "output" })],
      { key: "stepEditor.duplicateAttributeName", vars: { name: "param" } },
    ],
    [
      "invalid optional defaults",
      [
        attribute({
          role: "optional",
          name: "count",
          dataType: AttributeType.Number,
          defaultValue: "not-a-number",
        }),
      ],
      {
        key: "stepEditor.invalidDefaultValue",
        vars: { name: "count", reason: "validation.jsonInvalid" },
      },
    ],
    [
      "missing match languages",
      [
        attribute({
          name: "route",
          matchScript: "$.kind",
          matchLanguage: " ",
        }),
      ],
      { key: "stepEditor.matchLanguageRequired", vars: { name: "route" } },
    ],
    [
      "missing const defaults",
      [
        attribute({
          role: "const",
          name: "flag",
          dataType: AttributeType.Boolean,
        }),
      ],
      { key: "stepEditor.constDefaultRequired", vars: { name: "flag" } },
    ],
    [
      "optional attributes without defaults",
      [attribute({ role: "optional", name: "maybe" })],
      null,
    ],
  ];

  test.each(attributeValidationCases)(
    "validates %s",
    (_, attributes, expected) => {
      expect(validateAttributesList(attributes)).toEqual(expected);
    }
  );

  test("creates API attributes", () => {
    const result = createStepAttributes([
      attribute({
        name: "input",
        compensated: true,
        collect: "last",
        forEach: true,
        matchLanguage: " lua ",
        matchScript: ' return value == "email" ',
        mappingName: " request ",
        mappingLanguage: " jpath ",
        mappingScript: " $.payload ",
      }),
      attribute({
        id: "attr-2",
        role: "optional",
        name: "optional",
        dataType: AttributeType.Number,
        defaultValue: " 10 ",
        deadline: 3000,
      }),
      attribute({
        id: "attr-3",
        role: "const",
        name: "constant",
        defaultValue: ' "fixed" ',
      }),
      attribute({
        id: "attr-4",
        role: "meta",
        name: "metadata",
        metaKey: " request_id ",
      }),
      attribute({
        id: "attr-5",
        role: "output",
        name: "output",
        dataType: AttributeType.Object,
        mappingScript: " $.result ",
      }),
    ]);

    expect(result.input).toEqual({
      role: AttributeRole.Required,
      type: AttributeType.String,
      compensated: true,
      required: {
        collect: "last",
        for_each: true,
        match: { language: "lua", script: 'return value == "email"' },
        mapping: {
          name: "request",
          script: { language: "jpath", script: "$.payload" },
        },
      },
    });
    expect(result.optional.optional).toEqual({ default: "10", deadline: 3000 });
    expect(result.constant.const).toEqual({ value: '"fixed"' });
    expect(result.metadata.meta).toEqual({ key: "request_id" });
    expect(result.output.output?.mapping?.script).toEqual({
      language: "lua",
      script: "$.result",
    });
  });

  test.each([
    ["first collect", attribute({ collect: "first" }), "required", "collect"],
    ["false for-each", attribute({ forEach: false }), "required", "for_each"],
    ["blank match", attribute({ matchScript: "   " }), "required", "match"],
    [
      "blank optional default",
      attribute({ role: "optional", defaultValue: "   " }),
      "optional",
      "default",
    ],
  ])("omits %s", (_, item, configKey, valueKey) => {
    const spec = createStepAttributes([item])[item.name] as Record<string, any>;
    expect(spec[configKey]?.[valueKey]).toBeUndefined();
  });

  type ValidationArgs = Parameters<typeof getValidationError>[0];
  const baseValidationArgs: ValidationArgs = {
    isCreateMode: false,
    stepId: "step-1",
    attributes: [],
    stepType: "sync",
    script: "",
    endpoint: "https://example.com",
    httpMethod: "POST",
    httpTimeout: 5000,
    flowGoals: "",
    handling: "standard",
    compensateEndpoint: "",
  };
  const validationCases: Array<
    [string, Partial<ValidationArgs>, ReturnType<typeof getValidationError>]
  > = [
    [
      "missing create ID",
      { isCreateMode: true, stepId: " " },
      { key: "stepEditor.stepIdRequired" },
    ],
    ["blank edit ID", { stepId: " " }, null],
    [
      "invalid attributes",
      { attributes: [attribute({ name: " " })] },
      { key: "stepEditor.attributeNameRequired" },
    ],
    [
      "missing script",
      { stepType: "script", script: " " },
      { key: "stepEditor.scriptRequired" },
    ],
    [
      "missing flow goals",
      { stepType: "flow", flowGoals: " ", endpoint: "", httpTimeout: 0 },
      { key: "stepEditor.flowGoalsRequired" },
    ],
    [
      "valid flow",
      {
        stepType: "flow",
        flowGoals: "goal-a, goal-b",
        endpoint: "",
        httpTimeout: 0,
      },
      null,
    ],
    [
      "missing endpoint",
      { endpoint: " " },
      { key: "stepEditor.endpointRequired" },
    ],
    ["zero timeout", { httpTimeout: 0 }, { key: "stepEditor.timeoutPositive" }],
    [
      "negative timeout",
      { httpTimeout: -1000 },
      { key: "stepEditor.timeoutPositive" },
    ],
    ["valid HTTP configuration", { attributes: [attribute()] }, null],
    [
      "duplicate mapping names",
      {
        attributes: [
          attribute({ name: "a", mappingName: "shared" }),
          attribute({
            id: "attr-2",
            role: "optional",
            name: "b",
            mappingName: "shared",
          }),
        ],
      },
      { key: "stepEditor.duplicateMappingName", vars: { name: "shared" } },
    ],
    [
      "const mappings",
      {
        attributes: [
          attribute({
            role: "const",
            name: "constant",
            defaultValue: '"x"',
            mappingName: "illegal",
          }),
        ],
      },
      { key: "stepEditor.constMappingNotAllowed", vars: { name: "constant" } },
    ],
    [
      "missing mapping language",
      {
        attributes: [
          attribute({
            role: "output",
            name: "result",
            mappingScript: "$.result",
            mappingLanguage: " ",
          }),
        ],
      },
      { key: "stepEditor.mappingLanguageRequired", vars: { name: "result" } },
    ],
  ];

  test.each(validationCases)("reports %s", (_, overrides, expected) => {
    expect(getValidationError({ ...baseValidationArgs, ...overrides })).toEqual(
      expected
    );
  });
});
