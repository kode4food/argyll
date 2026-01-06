import {
  formatAttributeValue,
  getAttributeTooltipTitle,
  getAttributeValue,
} from "./attributeUtils";
import { ExecutionResult, AttributeRole, AttributeType } from "@/app/api";

describe("attributeUtils", () => {
  describe("formatAttributeValue", () => {
    it("formats null correctly", () => {
      expect(formatAttributeValue(null)).toBe("null");
    });

    it("formats undefined correctly", () => {
      expect(formatAttributeValue(undefined)).toBe("undefined");
    });

    it("formats strings with quotes", () => {
      expect(formatAttributeValue("hello")).toBe('"hello"');
    });

    it("formats numbers as strings", () => {
      expect(formatAttributeValue(42)).toBe("42");
      expect(formatAttributeValue(3.14)).toBe("3.14");
    });

    it("formats objects as formatted JSON", () => {
      const result = formatAttributeValue({ key: "value", nested: { a: 1 } });
      expect(result).toContain('"key"');
      expect(result).toContain('"value"');
      expect(result).toContain('"nested"');
    });

    it("formats arrays as JSON", () => {
      const result = formatAttributeValue([1, 2, 3]);
      expect(result).toContain("1");
      expect(result).toContain("2");
      expect(result).toContain("3");
    });

    it("handles objects that cannot be stringified", () => {
      const circular: any = { a: 1 };
      circular.self = circular;
      const result = formatAttributeValue(circular);
      expect(typeof result).toBe("string");
    });

    it("formats booleans correctly", () => {
      expect(formatAttributeValue(true)).toBe("true");
      expect(formatAttributeValue(false)).toBe("false");
    });
  });

  describe("getAttributeTooltipTitle", () => {
    it("returns 'Input Value' for required attributes", () => {
      expect(getAttributeTooltipTitle("required")).toBe("Input Value");
    });

    it("returns 'Output Value' for output attributes", () => {
      expect(getAttributeTooltipTitle("output")).toBe("Output Value");
    });

    it("returns 'Input Value' for optional attributes that were not defaulted", () => {
      expect(getAttributeTooltipTitle("optional", false)).toBe("Input Value");
    });

    it("returns 'Default Value' for optional attributes that were defaulted", () => {
      expect(getAttributeTooltipTitle("optional", true)).toBe("Default Value");
    });

    it("returns 'Input Value' for optional attributes without defaulted info", () => {
      expect(getAttributeTooltipTitle("optional")).toBe("Input Value");
    });
  });

  describe("getAttributeValue", () => {
    const mockExecution: ExecutionResult = {
      id: "exec-1",
      step_id: "step-1",
      status: "completed",
      inputs: { input1: "value1", input2: 42 },
      outputs: { output1: "result1" },
      duration_ms: 100,
      error_message: null,
    };

    it("extracts input values from flow state values", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Required,
          description: "",
        },
      };

      const attributeValues = {
        input1: { value: "value1", step: "step-0" },
      };
      const result = getAttributeValue(arg, mockExecution, attributeValues);
      expect(result.hasValue).toBe(true);
      expect(result.value).toBe("value1");
    });

    it("extracts output values from execution outputs", () => {
      const arg = {
        name: "output1",
        type: "string",
        argType: "output" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Output,
          description: "",
        },
      };

      const result = getAttributeValue(arg, mockExecution);
      expect(result.hasValue).toBe(true);
      expect(result.value).toBe("result1");
    });

    it("returns no value when attribute is not in inputs", () => {
      const arg = {
        name: "nonexistent",
        type: "string",
        argType: "required" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Required,
          description: "",
        },
      };

      const result = getAttributeValue(arg, mockExecution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("returns no value when execution is undefined", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Required,
          description: "",
        },
      };

      const result = getAttributeValue(arg, undefined);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("uses flow state values when execution inputs are missing", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Required,
          description: "",
        },
      };

      const emptyExecution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "skipped",
        inputs: {},
        outputs: undefined,
        duration_ms: 0,
        error_message: null,
      };

      const attributeValues = {
        input1: { value: "state-value", step: "step-0" },
      };

      const result = getAttributeValue(arg, emptyExecution, attributeValues);
      expect(result.hasValue).toBe(true);
      expect(result.value).toBe("state-value");
    });

    it("returns no value when execution has no inputs/outputs", () => {
      const emptyExecution: ExecutionResult = {
        id: "exec-1",
        step_id: "step-1",
        status: "completed",
        inputs: undefined,
        outputs: undefined,
        duration_ms: 0,
        error_message: null,
      };

      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          type: AttributeType.String,
          role: AttributeRole.Required,
          description: "",
        },
      };

      const result = getAttributeValue(arg, emptyExecution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("handles numeric output values", () => {
      const execWithNumericOutput: ExecutionResult = {
        ...mockExecution,
        outputs: { count: 123 },
      };

      const arg = {
        name: "count",
        type: "number",
        argType: "output" as const,
        spec: {
          type: AttributeType.Number,
          role: AttributeRole.Output,
          description: "",
        },
      };

      const result = getAttributeValue(arg, execWithNumericOutput);
      expect(result.hasValue).toBe(true);
      expect(result.value).toBe(123);
    });
  });
});
