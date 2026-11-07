import { getAttributeTooltipTitle, getAttributeValue } from "./attributeUtils";
import { ExecutionResult, AttributeRole } from "@/app/api";

describe("attributeUtils", () => {
  describe("getAttributeTooltipTitle", () => {
    it("returns key for required attributes", () => {
      expect(getAttributeTooltipTitle("required")).toBe("liveStep.inputValue");
    });

    it("returns key for output attributes", () => {
      expect(getAttributeTooltipTitle("output")).toBe("liveStep.outputValue");
    });

    it("returns key for feed attributes", () => {
      expect(getAttributeTooltipTitle("feed")).toBe("liveStep.feedValue");
    });

    it("returns key for const attributes", () => {
      expect(getAttributeTooltipTitle("const")).toBe("liveStep.defaultValue");
    });

    it("returns key for optional attributes that were not defaulted", () => {
      expect(getAttributeTooltipTitle("optional", false)).toBe(
        "liveStep.inputValue"
      );
    });

    it("returns key for optional attributes that were defaulted", () => {
      expect(getAttributeTooltipTitle("optional", true)).toBe(
        "liveStep.defaultValue"
      );
    });

    it("returns key for optional attributes without defaulted info", () => {
      expect(getAttributeTooltipTitle("optional")).toBe("liveStep.inputValue");
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

    it("does not use flow state arrays as input values", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          role: AttributeRole.Required,
          description: "",
        },
      };

      const execution = {
        ...mockExecution,
        inputs: {},
      };
      const result = getAttributeValue(arg, execution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("extracts output values from execution outputs", () => {
      const arg = {
        name: "output1",
        type: "string",
        argType: "output" as const,
        spec: {
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
          role: AttributeRole.Required,
          description: "",
        },
      };

      const result = getAttributeValue(arg, mockExecution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("treats empty flow state arrays as absent", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          role: AttributeRole.Required,
          description: "",
        },
      };
      const execution = {
        ...mockExecution,
        inputs: {},
      };
      const result = getAttributeValue(arg, execution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("does not use optional default when execution inputs are missing", () => {
      const arg = {
        name: "opt1",
        type: "string",
        argType: "optional" as const,
        spec: {
          role: AttributeRole.Optional,
          optional: { default: "fallback value" },
          description: "",
        },
      };
      const execution: ExecutionResult = {
        ...mockExecution,
        status: "skipped",
        inputs: {},
      };
      const result = getAttributeValue(arg, execution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("extracts input values by mapped input name", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          role: AttributeRole.Required,
          required: { mapping: { name: "mapped_input" } },
          description: "",
        },
      };
      const execution = {
        ...mockExecution,
        inputs: { mapped_input: "mapped value" },
      };

      const result = getAttributeValue(arg, execution);
      expect(result.hasValue).toBe(true);
      expect(result.value).toBe("mapped value");
    });

    it("returns no value when execution is undefined", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
          role: AttributeRole.Required,
          description: "",
        },
      };

      const result = getAttributeValue(arg, undefined);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
    });

    it("does not use flow state values when execution inputs are missing", () => {
      const arg = {
        name: "input1",
        type: "string",
        argType: "required" as const,
        spec: {
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

      const result = getAttributeValue(arg, emptyExecution);
      expect(result.hasValue).toBe(false);
      expect(result.value).toBeUndefined();
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
