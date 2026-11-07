import {
  isDefaultValue,
  parseState,
  filterDefaultValues,
  addRequiredDefaults,
} from "./stateUtils";
import { AttributeRole, ExecutionPlan } from "@/app/api";

describe("stateUtils", () => {
  describe("isDefaultValue", () => {
    it("returns true for empty string", () => {
      expect(isDefaultValue("")).toBe(true);
    });

    it("returns true for null", () => {
      expect(isDefaultValue(null)).toBe(true);
    });

    it("returns true for false", () => {
      expect(isDefaultValue(false)).toBe(true);
    });

    it("returns true for 0", () => {
      expect(isDefaultValue(0)).toBe(true);
    });

    it("returns true for empty object", () => {
      expect(isDefaultValue({})).toBe(true);
    });

    it("returns true for empty array", () => {
      expect(isDefaultValue([])).toBe(true);
    });

    it("returns false for non-empty values", () => {
      expect(isDefaultValue("hello")).toBe(false);
      expect(isDefaultValue(true)).toBe(false);
      expect(isDefaultValue(42)).toBe(false);
      expect(isDefaultValue({ a: 1 })).toBe(false);
      expect(isDefaultValue([1])).toBe(false);
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
    it("filters default values", () => {
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
        emptyArr: [],
        realString: "hello",
        realNumber: 42,
        realBool: true,
      });
    });
  });

  describe("addRequiredDefaults", () => {
    it("adds blank placeholders for required attributes", () => {
      const state = {};

      const executionPlan: ExecutionPlan = {
        goals: { steps: ["step1"] },
        required: ["boolProp", "numberProp", "stringProp"],
        steps: {
          step1: {
            id: "step1",
            name: "Step 1",
            type: "service",
            attributes: {
              boolProp: {
                role: AttributeRole.Required,
              },
              numberProp: {
                role: AttributeRole.Required,
              },
              stringProp: {
                role: AttributeRole.Required,
              },
            },
          },
        },
        attributes: {},
      };

      const result = addRequiredDefaults(state, executionPlan);

      expect(result).toEqual({
        boolProp: "",
        numberProp: "",
        stringProp: "",
      });
    });

    it("does not overwrite existing values", () => {
      const state = {
        boolProp: true,
        numberProp: 42,
      };

      const executionPlan: ExecutionPlan = {
        goals: { steps: ["step1"] },
        required: ["boolProp", "numberProp", "stringProp"],
        steps: {
          step1: {
            id: "step1",
            name: "Step 1",
            type: "service",
            attributes: {
              boolProp: {
                role: AttributeRole.Required,
              },
              numberProp: {
                role: AttributeRole.Required,
              },
              stringProp: {
                role: AttributeRole.Required,
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
