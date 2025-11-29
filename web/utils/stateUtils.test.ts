import {
  getDefaultValueForType,
  isDefaultValue,
  isUntypedDefaultValue,
  parseState,
  filterDefaultValues,
  addRequiredDefaults,
} from "./stateUtils";
import { AttributeType, Step, ExecutionPlan } from "@/app/api";

describe("stateUtils", () => {
  describe("getDefaultValueForType", () => {
    it("returns false for Boolean type", () => {
      expect(getDefaultValueForType(AttributeType.Boolean)).toBe(false);
    });

    it("returns 0 for Number type", () => {
      expect(getDefaultValueForType(AttributeType.Number)).toBe(0);
    });

    it("returns empty string for String type", () => {
      expect(getDefaultValueForType(AttributeType.String)).toBe("");
    });

    it("returns empty object for Object type", () => {
      expect(getDefaultValueForType(AttributeType.Object)).toEqual({});
    });

    it("returns empty array for Array type", () => {
      expect(getDefaultValueForType(AttributeType.Array)).toEqual([]);
    });

    it("returns null for Null type", () => {
      expect(getDefaultValueForType(AttributeType.Null)).toBe(null);
    });

    it("returns empty string for Any type", () => {
      expect(getDefaultValueForType(AttributeType.Any)).toBe("");
    });

    it("returns empty string for undefined type", () => {
      expect(getDefaultValueForType(undefined)).toBe("");
    });
  });

  describe("isDefaultValue", () => {
    it("returns true for false with Boolean type", () => {
      expect(isDefaultValue(false, AttributeType.Boolean)).toBe(true);
    });

    it("returns false for true with Boolean type", () => {
      expect(isDefaultValue(true, AttributeType.Boolean)).toBe(false);
    });

    it("returns true for 0 with Number type", () => {
      expect(isDefaultValue(0, AttributeType.Number)).toBe(true);
    });

    it("returns false for 42 with Number type", () => {
      expect(isDefaultValue(42, AttributeType.Number)).toBe(false);
    });

    it("returns true for empty string with String type", () => {
      expect(isDefaultValue("", AttributeType.String)).toBe(true);
    });

    it("returns false for non-empty string with String type", () => {
      expect(isDefaultValue("hello", AttributeType.String)).toBe(false);
    });

    it("returns true for empty object with Object type", () => {
      expect(isDefaultValue({}, AttributeType.Object)).toBe(true);
    });

    it("returns false for non-empty object with Object type", () => {
      expect(isDefaultValue({ a: 1 }, AttributeType.Object)).toBe(false);
    });

    it("returns true for empty array with Array type", () => {
      expect(isDefaultValue([], AttributeType.Array)).toBe(true);
    });

    it("returns false for non-empty array with Array type", () => {
      expect(isDefaultValue([1, 2], AttributeType.Array)).toBe(false);
    });

    it("returns true for null with Null type", () => {
      expect(isDefaultValue(null, AttributeType.Null)).toBe(true);
    });

    it("returns false for false with Number type (wrong type)", () => {
      expect(isDefaultValue(false, AttributeType.Number)).toBe(false);
    });

    it("returns false for 0 with Boolean type (wrong type)", () => {
      expect(isDefaultValue(0, AttributeType.Boolean)).toBe(false);
    });

    it("falls back to untyped check when type is undefined", () => {
      expect(isDefaultValue("", undefined)).toBe(true);
      expect(isDefaultValue(false, undefined)).toBe(true);
      expect(isDefaultValue(0, undefined)).toBe(true);
      expect(isDefaultValue({}, undefined)).toBe(true);
      expect(isDefaultValue([], undefined)).toBe(true);
    });
  });

  describe("isUntypedDefaultValue", () => {
    it("returns true for empty string", () => {
      expect(isUntypedDefaultValue("")).toBe(true);
    });

    it("returns true for null", () => {
      expect(isUntypedDefaultValue(null)).toBe(true);
    });

    it("returns true for false", () => {
      expect(isUntypedDefaultValue(false)).toBe(true);
    });

    it("returns true for 0", () => {
      expect(isUntypedDefaultValue(0)).toBe(true);
    });

    it("returns true for empty object", () => {
      expect(isUntypedDefaultValue({})).toBe(true);
    });

    it("returns true for empty array", () => {
      expect(isUntypedDefaultValue([])).toBe(true);
    });

    it("returns false for non-empty values", () => {
      expect(isUntypedDefaultValue("hello")).toBe(false);
      expect(isUntypedDefaultValue(true)).toBe(false);
      expect(isUntypedDefaultValue(42)).toBe(false);
      expect(isUntypedDefaultValue({ a: 1 })).toBe(false);
      expect(isUntypedDefaultValue([1])).toBe(false);
    });
  });

  describe("parseState", () => {
    it("parses valid JSON", () => {
      expect(parseState('{"foo": "bar"}')).toEqual({ foo: "bar" });
    });

    it("returns empty object for invalid JSON", () => {
      expect(parseState("not json")).toEqual({});
    });

    it("returns empty object for empty string", () => {
      expect(parseState("")).toEqual({});
    });
  });

  describe("filterDefaultValues", () => {
    it("filters out default values without type info", () => {
      const state = {
        empty: "",
        zero: 0,
        falseBool: false,
        emptyObj: {},
        emptyArr: [],
        nullVal: null,
        realString: "hello",
        realNumber: 42,
        realBool: true,
      };

      const filtered = filterDefaultValues(state);

      expect(filtered).toEqual({
        realString: "hello",
        realNumber: 42,
        realBool: true,
      });
    });

    it("filters out type-specific defaults with step info", () => {
      const state = {
        boolProp: false,
        numberProp: 0,
        stringProp: "",
        realBool: true,
        realNumber: 42,
      };

      const steps: Step[] = [
        {
          id: "step1",
          name: "Step 1",
          type: "sync",
          version: "1.0",
          attributes: {
            boolProp: {
              role: 0,
              type: AttributeType.Boolean,
            },
            numberProp: {
              role: 0,
              type: AttributeType.Number,
            },
            stringProp: {
              role: 0,
              type: AttributeType.String,
            },
          },
        },
      ];

      const filtered = filterDefaultValues(state, steps);

      expect(filtered).toEqual({
        realBool: true,
        realNumber: 42,
      });
    });

    it("keeps false for number type (wrong default)", () => {
      const state = {
        numberProp: false,
      };

      const steps: Step[] = [
        {
          id: "step1",
          name: "Step 1",
          type: "sync",
          version: "1.0",
          attributes: {
            numberProp: {
              role: 0,
              type: AttributeType.Number,
            },
          },
        },
      ];

      const filtered = filterDefaultValues(state, steps);

      expect(filtered).toEqual({ numberProp: false });
    });
  });

  describe("addRequiredDefaults", () => {
    it("adds typed defaults for required attributes", () => {
      const state = {};

      const executionPlan: ExecutionPlan = {
        goals: ["step1"],
        required: ["boolProp", "numberProp", "stringProp"],
        steps: {
          step1: {
            id: "step1",
            name: "Step 1",
            type: "sync",
            version: "1.0",
            attributes: {
              boolProp: {
                role: 0,
                type: AttributeType.Boolean,
              },
              numberProp: {
                role: 0,
                type: AttributeType.Number,
              },
              stringProp: {
                role: 0,
                type: AttributeType.String,
              },
            },
          },
        },
        attributes: {},
      };

      const result = addRequiredDefaults(state, executionPlan);

      expect(result).toEqual({
        boolProp: false,
        numberProp: 0,
        stringProp: "",
      });
    });

    it("does not overwrite existing values", () => {
      const state = {
        boolProp: true,
        numberProp: 42,
      };

      const executionPlan: ExecutionPlan = {
        goals: ["step1"],
        required: ["boolProp", "numberProp", "stringProp"],
        steps: {
          step1: {
            id: "step1",
            name: "Step 1",
            type: "sync",
            version: "1.0",
            attributes: {
              boolProp: {
                role: 0,
                type: AttributeType.Boolean,
              },
              numberProp: {
                role: 0,
                type: AttributeType.Number,
              },
              stringProp: {
                role: 0,
                type: AttributeType.String,
              },
            },
          },
        },
        attributes: {},
      };

      const result = addRequiredDefaults(state, executionPlan);

      expect(result).toEqual({
        boolProp: true,
        numberProp: 42,
        stringProp: "",
      });
    });
  });
});
