import {
  getStepType,
  getStepTypeLabel,
  sortStepsByType,
  validateDefaultValue,
  getSortedAttributes,
} from "./stepUtils";
import { Step, AttributeRole, AttributeType } from "@/app/api";

describe("stepUtils", () => {
  describe("getSortedAttributes", () => {
    test("returns attributes sorted by name within each role", () => {
      const attributes = {
        zebra: { role: AttributeRole.Required, type: AttributeType.String },
        alpha: { role: AttributeRole.Required, type: AttributeType.String },
        beta: { role: AttributeRole.Optional, type: AttributeType.Number },
        gamma: { role: AttributeRole.Output, type: AttributeType.String },
        delta: { role: AttributeRole.Output, type: AttributeType.Boolean },
      };

      const result = getSortedAttributes(attributes);

      expect(result).toHaveLength(5);
      // Required first
      expect(result[0].name).toBe("alpha");
      expect(result[1].name).toBe("zebra");
      // Optional second
      expect(result[2].name).toBe("beta");
      // Outputs last
      expect(result[3].name).toBe("delta");
      expect(result[4].name).toBe("gamma");
    });

    test("handles empty attributes object", () => {
      const result = getSortedAttributes({});
      expect(result).toEqual([]);
    });

    test("handles only required attributes", () => {
      const attributes = {
        input2: { role: AttributeRole.Required, type: AttributeType.String },
        input1: { role: AttributeRole.Required, type: AttributeType.Number },
      };

      const result = getSortedAttributes(attributes);

      expect(result).toHaveLength(2);
      expect(result[0].name).toBe("input1");
      expect(result[1].name).toBe("input2");
    });

    test("handles only optional attributes", () => {
      const attributes = {
        opt2: { role: AttributeRole.Optional, type: AttributeType.String },
        opt1: { role: AttributeRole.Optional, type: AttributeType.Number },
      };

      const result = getSortedAttributes(attributes);

      expect(result).toHaveLength(2);
      expect(result[0].name).toBe("opt1");
      expect(result[1].name).toBe("opt2");
    });

    test("handles only output attributes", () => {
      const attributes = {
        result2: { role: AttributeRole.Output, type: AttributeType.String },
        result1: { role: AttributeRole.Output, type: AttributeType.Number },
      };

      const result = getSortedAttributes(attributes);

      expect(result).toHaveLength(2);
      expect(result[0].name).toBe("result1");
      expect(result[1].name).toBe("result2");
    });

    test("preserves attribute spec objects", () => {
      const attributes = {
        input: { role: AttributeRole.Required, type: AttributeType.String },
      };

      const result = getSortedAttributes(attributes);

      expect(result[0]).toEqual({
        name: "input",
        spec: { role: AttributeRole.Required, type: AttributeType.String },
      });
    });
  });

  describe("getStepType", () => {
    test("returns resolver when step has outputs but no inputs", () => {
      const step: Step = {
        id: "step-1",
        name: "Test Step",
        type: "sync",
        attributes: {
          result: { role: AttributeRole.Output, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("resolver");
    });

    test("returns collector when step has inputs but no outputs", () => {
      const step: Step = {
        id: "step-2",
        name: "Test Step",
        type: "async",
        attributes: {
          input1: { role: AttributeRole.Required, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("collector");
    });

    test("returns processor when step has both inputs and outputs", () => {
      const step: Step = {
        id: "step-3",
        name: "Test Step",
        type: "sync",
        attributes: {
          input1: { role: AttributeRole.Required, type: AttributeType.String },
          result: { role: AttributeRole.Output, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("processor");
    });

    test("returns neutral when step has no inputs and no outputs", () => {
      const step: Step = {
        id: "step-4",
        name: "Test Step",
        type: "script",
        attributes: {},
        script: {
          language: "ale",
          script: "{:result 1}",
        },
      };

      expect(getStepType(step)).toBe("neutral");
    });

    test("optional args alone do not affect step type (resolver remains resolver)", () => {
      const step: Step = {
        id: "step-5",
        name: "Test Step",
        type: "sync",
        attributes: {
          opt1: { role: AttributeRole.Optional, type: AttributeType.String },
          result: { role: AttributeRole.Output, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("resolver");
    });

    test("optional args alone do not affect step type (neutral remains neutral)", () => {
      const step: Step = {
        id: "step-6",
        name: "Test Step",
        type: "async",
        attributes: {
          opt1: { role: AttributeRole.Optional, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("neutral");
    });

    test("handles step with multiple required and optional args", () => {
      const step: Step = {
        id: "step-7",
        name: "Test Step",
        type: "sync",
        attributes: {
          req1: { role: AttributeRole.Required, type: AttributeType.String },
          req2: { role: AttributeRole.Required, type: AttributeType.Number },
          opt1: { role: AttributeRole.Optional, type: AttributeType.String },
          opt2: { role: AttributeRole.Optional, type: AttributeType.Boolean },
          result1: { role: AttributeRole.Output, type: AttributeType.String },
          result2: { role: AttributeRole.Output, type: AttributeType.String },
        },
        http: {
          endpoint: "http://test",
          timeout: 1000,
        },
      };

      expect(getStepType(step)).toBe("processor");
    });

    test("returns neutral when step has empty attributes", () => {
      const step: Step = {
        id: "step-8",
        name: "Test Step",
        type: "script",
        attributes: {},
        script: {
          language: "ale",
          script: "{:result 1}",
        },
      };

      expect(getStepType(step)).toBe("neutral");
    });
  });

  describe("getStepTypeLabel", () => {
    test("returns correct label for resolver", () => {
      expect(getStepTypeLabel("resolver")).toBe("R");
    });

    test("returns correct label for collector", () => {
      expect(getStepTypeLabel("collector")).toBe("C");
    });

    test("returns correct label for processor", () => {
      expect(getStepTypeLabel("processor")).toBe("P");
    });

    test("returns correct label for neutral", () => {
      expect(getStepTypeLabel("neutral")).toBe("S");
    });
  });

  describe("sortStepsByType", () => {
    const createStep = (
      id: string,
      name: string,
      hasInputs: boolean,
      hasOutputs: boolean
    ): Step => ({
      id,
      name,
      type: "sync",
      attributes: {
        ...(hasInputs
          ? {
              input: {
                role: AttributeRole.Required,
                type: AttributeType.String,
              },
            }
          : {}),
        ...(hasOutputs
          ? {
              output: {
                role: AttributeRole.Output,
                type: AttributeType.String,
              },
            }
          : {}),
      },
      http: {
        endpoint: "http://test",
        timeout: 1000,
      },
    });

    test("sorts steps by type priority", () => {
      const steps: Step[] = [
        createStep("1", "Neutral Step", false, false),
        createStep("2", "Resolver Step", false, true),
        createStep("3", "Collector Step", true, false),
        createStep("4", "Processor Step", true, true),
      ];

      const sorted = sortStepsByType(steps);

      expect(sorted[0].id).toBe("3");
      expect(sorted[1].id).toBe("4");
      expect(sorted[2].id).toBe("2");
      expect(sorted[3].id).toBe("1");
    });

    test("sorts steps alphabetically within same type", () => {
      const steps: Step[] = [
        createStep("1", "Zebra Processor", true, true),
        createStep("2", "Alpha Processor", true, true),
        createStep("3", "Beta Processor", true, true),
      ];

      const sorted = sortStepsByType(steps);

      expect(sorted[0].name).toBe("Alpha Processor");
      expect(sorted[1].name).toBe("Beta Processor");
      expect(sorted[2].name).toBe("Zebra Processor");
    });

    test("handles mixed types with alphabetical sorting", () => {
      const steps: Step[] = [
        createStep("1", "Z Resolver", false, true),
        createStep("2", "A Collector", true, false),
        createStep("3", "B Collector", true, false),
        createStep("4", "A Resolver", false, true),
      ];

      const sorted = sortStepsByType(steps);

      expect(sorted[0].name).toBe("A Collector");
      expect(sorted[1].name).toBe("B Collector");
      expect(sorted[2].name).toBe("A Resolver");
      expect(sorted[3].name).toBe("Z Resolver");
    });

    test("does not mutate original array", () => {
      const steps: Step[] = [
        createStep("1", "B Step", false, true),
        createStep("2", "A Step", false, true),
      ];
      const originalOrder = steps.map((s) => s.id);

      sortStepsByType(steps);

      expect(steps.map((s) => s.id)).toEqual(originalOrder);
    });

    test("handles empty array", () => {
      const sorted = sortStepsByType([]);
      expect(sorted).toEqual([]);
    });

    test("handles single step", () => {
      const steps: Step[] = [createStep("1", "Single Step", true, true)];
      const sorted = sortStepsByType(steps);
      expect(sorted).toHaveLength(1);
      expect(sorted[0].id).toBe("1");
    });
  });

  describe("validateDefaultValue", () => {
    describe("String type", () => {
      test("accepts valid JSON string", () => {
        const result = validateDefaultValue(
          '"hello world"',
          AttributeType.String
        );
        expect(result.valid).toBe(true);
      });

      test("rejects unquoted string", () => {
        const result = validateDefaultValue("hello", AttributeType.String);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });

      test("rejects number as string", () => {
        const result = validateDefaultValue("42", AttributeType.String);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonString");
      });
    });

    describe("Number type", () => {
      test("accepts valid integer", () => {
        const result = validateDefaultValue("42", AttributeType.Number);
        expect(result.valid).toBe(true);
      });

      test("accepts valid float", () => {
        const result = validateDefaultValue("3.14", AttributeType.Number);
        expect(result.valid).toBe(true);
      });

      test("rejects non-number", () => {
        const result = validateDefaultValue(
          '"not a number"',
          AttributeType.Number
        );
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonNumber");
      });

      test("rejects invalid JSON", () => {
        const result = validateDefaultValue("abc", AttributeType.Number);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });
    });

    describe("Boolean type", () => {
      test("accepts true", () => {
        const result = validateDefaultValue("true", AttributeType.Boolean);
        expect(result.valid).toBe(true);
      });

      test("accepts false", () => {
        const result = validateDefaultValue("false", AttributeType.Boolean);
        expect(result.valid).toBe(true);
      });

      test("rejects string 'yes'", () => {
        const result = validateDefaultValue('"yes"', AttributeType.Boolean);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonBoolean");
      });
    });

    describe("Object type", () => {
      test("accepts valid JSON object", () => {
        const result = validateDefaultValue(
          '{"key": "value"}',
          AttributeType.Object
        );
        expect(result.valid).toBe(true);
      });

      test("rejects array", () => {
        const result = validateDefaultValue("[1, 2, 3]", AttributeType.Object);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonObject");
      });

      test("rejects malformed JSON", () => {
        const result = validateDefaultValue(
          "{key: value}",
          AttributeType.Object
        );
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });
    });

    describe("Array type", () => {
      test("accepts valid JSON array", () => {
        const result = validateDefaultValue("[1, 2, 3]", AttributeType.Array);
        expect(result.valid).toBe(true);
      });

      test("rejects object", () => {
        const result = validateDefaultValue(
          '{"key": "value"}',
          AttributeType.Array
        );
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonArray");
      });

      test("rejects malformed JSON", () => {
        const result = validateDefaultValue("[1, 2, 3", AttributeType.Array);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });
    });

    describe("Null type", () => {
      test("accepts null", () => {
        const result = validateDefaultValue("null", AttributeType.Null);
        expect(result.valid).toBe(true);
      });

      test("rejects 'nil'", () => {
        const result = validateDefaultValue("nil", AttributeType.Null);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });

      test("rejects string 'null'", () => {
        const result = validateDefaultValue('"null"', AttributeType.Null);
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonNull");
      });
    });

    describe("Any type", () => {
      test("accepts valid JSON string", () => {
        const result = validateDefaultValue('"whatever"', AttributeType.Any);
        expect(result.valid).toBe(true);
      });

      test("accepts valid JSON number", () => {
        const result = validateDefaultValue("42", AttributeType.Any);
        expect(result.valid).toBe(true);
      });

      test("accepts valid JSON object", () => {
        const result = validateDefaultValue(
          '{"key":"value"}',
          AttributeType.Any
        );
        expect(result.valid).toBe(true);
      });

      test("accepts valid JSON array", () => {
        const result = validateDefaultValue("[1,2,3]", AttributeType.Any);
        expect(result.valid).toBe(true);
      });

      test("rejects invalid JSON", () => {
        const result = validateDefaultValue(
          "not valid json",
          AttributeType.Any
        );
        expect(result.valid).toBe(false);
        expect(result.errorKey).toBe("validation.jsonInvalid");
      });
    });

    describe("Empty values", () => {
      test("accepts empty string for all types", () => {
        expect(validateDefaultValue("", AttributeType.String).valid).toBe(true);
        expect(validateDefaultValue("", AttributeType.Number).valid).toBe(true);
        expect(validateDefaultValue("", AttributeType.Boolean).valid).toBe(
          true
        );
        expect(validateDefaultValue("", AttributeType.Object).valid).toBe(true);
        expect(validateDefaultValue("", AttributeType.Array).valid).toBe(true);
        expect(validateDefaultValue("", AttributeType.Null).valid).toBe(true);
        expect(validateDefaultValue("", AttributeType.Any).valid).toBe(true);
      });

      test("accepts whitespace-only string for all types", () => {
        expect(validateDefaultValue("   ", AttributeType.String).valid).toBe(
          true
        );
        expect(validateDefaultValue("   ", AttributeType.Number).valid).toBe(
          true
        );
      });
    });
  });
});
