import {
  getAttributeModifiers,
  getSortedAttributes,
  getSpecPreview,
  getStepType,
  sortStepsByType,
  validateDefaultValue,
} from "./stepUtils";
import { AttributeRole, AttributeSpec, Step } from "@/app/api";
import {
  IconArrayMultiple,
  IconAttributeMatch,
  IconDuration,
  IconMapping,
} from "@/utils/iconRegistry";

const attr = (
  role: AttributeRole,
  config: Partial<AttributeSpec> = {}
): AttributeSpec => ({ role, ...config });

const step = (
  id: string,
  name: string,
  attributes: Record<string, AttributeSpec> = {}
): Step => ({ id, name, type: "service", attributes });

describe("stepUtils", () => {
  test("sorts attributes by role then name without replacing specs", () => {
    const alpha = attr(AttributeRole.Required);
    const attributes = {
      zebra: attr(AttributeRole.Required),
      alpha,
      const1: attr(AttributeRole.Const),
      optional1: attr(AttributeRole.Optional),
      meta1: attr(AttributeRole.Meta),
      feed1: attr(AttributeRole.Feed),
      zOutput: attr(AttributeRole.Output),
      aOutput: attr(AttributeRole.Output),
    };

    const result = getSortedAttributes(attributes);

    expect(result.map(({ name }) => name)).toEqual([
      "alpha",
      "zebra",
      "const1",
      "optional1",
      "meta1",
      "feed1",
      "aOutput",
      "zOutput",
    ]);
    expect(result[0].spec).toBe(alpha);
    expect(getSortedAttributes({})).toEqual([]);
  });

  test.each([
    ["resolver", { output: attr(AttributeRole.Output) }],
    ["collector", { input: attr(AttributeRole.Required) }],
    [
      "processor",
      {
        input: attr(AttributeRole.Required),
        output: attr(AttributeRole.Output),
        optional: attr(AttributeRole.Optional),
      },
    ],
    [
      "resolver",
      {
        optional: attr(AttributeRole.Optional),
        output: attr(AttributeRole.Output),
      },
    ],
    ["standalone", { optional: attr(AttributeRole.Optional) }],
    ["standalone", {}],
  ])("classifies a step as %s", (expected, attributes) => {
    expect(getStepType(step("id", "Step", attributes))).toBe(expected);
  });

  test("classifies a step without attributes as standalone", () => {
    expect(
      getStepType({ id: "id", name: "Step", type: "service" } as Step)
    ).toBe("standalone");
  });

  test("sorts steps by type and name without mutation", () => {
    const steps = [
      step("standalone", "Neutral"),
      step("resolver-z", "Z Resolver", {
        output: attr(AttributeRole.Output),
      }),
      step("collector-b", "B Collector", {
        input: attr(AttributeRole.Required),
      }),
      step("processor", "Processor", {
        input: attr(AttributeRole.Required),
        output: attr(AttributeRole.Output),
      }),
      step("collector-a", "A Collector", {
        input: attr(AttributeRole.Required),
      }),
      step("resolver-a", "A Resolver", {
        output: attr(AttributeRole.Output),
      }),
    ];
    const original = [...steps];

    expect(sortStepsByType(steps).map(({ id }) => id)).toEqual([
      "collector-a",
      "collector-b",
      "processor",
      "resolver-a",
      "resolver-z",
      "standalone",
    ]);
    expect(steps).toEqual(original);
    expect(sortStepsByType([])).toEqual([]);
    expect(sortStepsByType([steps[0]])).toEqual([steps[0]]);
  });

  test.each([
    ["none", attr(AttributeRole.Required), []],
    [
      "deadline",
      attr(AttributeRole.Optional, { optional: { deadline: 1000 } }),
      [{ kind: "icon", Icon: IconDuration }],
    ],
    [
      "zero deadline",
      attr(AttributeRole.Optional, { optional: { deadline: 0 } }),
      [],
    ],
    [
      "match",
      attr(AttributeRole.Required, {
        required: { match: { language: "argyll", script: "value.kind" } },
      }),
      [
        {
          kind: "match",
          Icon: IconAttributeMatch,
          script: { language: "argyll", script: "value.kind" },
        },
      ],
    ],
    [
      "required mapping",
      attr(AttributeRole.Required, { required: { mapping: { name: "in" } } }),
      [{ kind: "icon", Icon: IconMapping }],
    ],
    [
      "optional mapping",
      attr(AttributeRole.Optional, { optional: { mapping: { name: "in" } } }),
      [{ kind: "icon", Icon: IconMapping }],
    ],
    [
      "output mapping",
      attr(AttributeRole.Output, { output: { mapping: { name: "out" } } }),
      [{ kind: "icon", Icon: IconMapping }],
    ],
    [
      "required collect",
      attr(AttributeRole.Required, { required: { collect: "all" } }),
      [{ kind: "collect", collect: "all" }],
    ],
    [
      "optional collect",
      attr(AttributeRole.Optional, { optional: { collect: "some" } }),
      [{ kind: "collect", collect: "some" }],
    ],
    [
      "first collect",
      attr(AttributeRole.Required, { required: { collect: "first" } }),
      [],
    ],
    [
      "required for-each",
      attr(AttributeRole.Required, { required: { for_each: true } }),
      [{ kind: "icon", Icon: IconArrayMultiple }],
    ],
    [
      "optional for-each",
      attr(AttributeRole.Optional, { optional: { for_each: true } }),
      [{ kind: "icon", Icon: IconArrayMultiple }],
    ],
  ])("returns %s modifiers", (_, spec, expected) => {
    expect(getAttributeModifiers(spec)).toEqual(expected);
  });

  test("returns modifiers in pipeline order", () => {
    expect(
      getAttributeModifiers(
        attr(AttributeRole.Required, {
          required: {
            match: { language: "argyll", script: "value" },
            mapping: { name: "mapped" },
            collect: "all",
            for_each: true,
          },
        })
      ).map(({ kind }) => kind)
    ).toEqual(["match", "icon", "collect", "icon"]);

    expect(
      getAttributeModifiers(
        attr(AttributeRole.Optional, {
          optional: {
            deadline: 100,
            mapping: { name: "mapped" },
            collect: "last",
            for_each: true,
          },
        })
      ).map(({ kind }) => kind)
    ).toEqual(["icon", "icon", "collect", "icon"]);
  });

  test.each([
    ['"hello"', true, undefined],
    ["42", true, undefined],
    ["3.14", true, undefined],
    ["true", true, undefined],
    ['{"a":1}', true, undefined],
    ["[]", true, undefined],
    ["null", true, undefined],
    ["hello", false, "validation.jsonInvalid"],
    ["abc", false, "validation.jsonInvalid"],
    ["{bad", false, "validation.jsonInvalid"],
    ["[bad", false, "validation.jsonInvalid"],
    ["nil", false, "validation.jsonInvalid"],
  ])("validates value %s", (value, valid, errorKey) => {
    expect(validateDefaultValue(value)).toEqual(
      errorKey ? { valid, errorKey } : { valid }
    );
  });

  test("allows empty values", () => {
    expect(validateDefaultValue("")).toEqual({ valid: true });
    expect(validateDefaultValue("   ")).toEqual({ valid: true });
  });

  test.each([
    [{ role: AttributeRole.Required }, undefined],
    [{ role: AttributeRole.Output }, undefined],
    [{ role: AttributeRole.Optional }, undefined],
    [{ role: AttributeRole.Optional, optional: { default: "1" } }, "1"],
    [{ role: AttributeRole.Const, const: { value: '"x"' } }, '"x"'],
    [{ role: AttributeRole.Meta, meta: { key: "flow_id" } }, "flow_id"],
    [{ role: AttributeRole.Feed, feed: { id: "total" } }, "total"],
  ] as [AttributeSpec, string | undefined][])(
    "previews %j as %s",
    (spec, expected) => {
      expect(getSpecPreview(spec)).toBe(expected);
    }
  );
});
