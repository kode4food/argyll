import {
  Attribute,
  buildAttributesFromStep,
  validateAttributesList,
  createStepAttributes,
  getAttributeIconProps,
  getValidationError,
} from "./stepEditorUtils";
import { AttributeRole, AttributeType, Step } from "@/app/api";

describe("stepEditorUtils", () => {
  describe("buildAttributesFromStep", () => {
    it("returns empty array for null step", () => {
      expect(buildAttributesFromStep(null)).toEqual([]);
    });

    it("converts step attributes to Attribute objects", () => {
      const step: Step = {
        id: "test-step",
        name: "Test",
        type: "sync",
        attributes: {
          required_arg: {
            role: AttributeRole.Required,
            type: AttributeType.String,
            mapping: { name: "child_in" },
            description: "",
          },
          const_arg: {
            role: AttributeRole.Const,
            type: AttributeType.String,
            default: '"fixed"',
            description: "",
          },
          optional_arg: {
            role: AttributeRole.Optional,
            type: AttributeType.Number,
            default: 42,
            timeout: 3000,
            description: "",
          },
          output_arg: {
            role: AttributeRole.Output,
            type: AttributeType.String,
            mapping: { name: "child_out" },
            description: "",
          },
        },
        flow: { goals: ["goal-1"] },
      };

      const result = buildAttributesFromStep(step);

      expect(result).toHaveLength(4);

      const inputAttrs = result.filter((a) => a.attrType === "input");
      const constAttrs = result.filter((a) => a.attrType === "const");
      const optionalAttrs = result.filter((a) => a.attrType === "optional");
      const outputAttrs = result.filter((a) => a.attrType === "output");

      expect(inputAttrs).toHaveLength(1);
      expect(inputAttrs[0].name).toBe("required_arg");

      expect(constAttrs).toHaveLength(1);
      expect(constAttrs[0].name).toBe("const_arg");
      expect(constAttrs[0].defaultValue).toBe('"fixed"');

      expect(optionalAttrs).toHaveLength(1);
      expect(optionalAttrs[0].name).toBe("optional_arg");
      expect(optionalAttrs[0].defaultValue).toBe("42");
      expect(optionalAttrs[0].timeout).toBe(3000);

      expect(outputAttrs).toHaveLength(1);
      expect(outputAttrs[0].name).toBe("output_arg");
      expect(inputAttrs[0].mappingName).toBe("child_in");
      expect(outputAttrs[0].mappingName).toBe("child_out");
    });
  });

  describe("validateAttributesList", () => {
    it("returns null for valid attributes", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "param1",
          dataType: AttributeType.String,
        },
        {
          id: "attr-2",
          attrType: "output",
          name: "result",
          dataType: AttributeType.String,
        },
      ];

      expect(validateAttributesList(attributes)).toBeNull();
    });

    it("detects empty attribute names", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "   ",
          dataType: AttributeType.String,
        },
      ];

      expect(validateAttributesList(attributes)).toEqual({
        key: "stepEditor.attributeNameRequired",
      });
    });

    it("detects duplicate attribute names", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "param",
          dataType: AttributeType.String,
        },
        {
          id: "attr-2",
          attrType: "output",
          name: "param",
          dataType: AttributeType.String,
        },
      ];

      expect(validateAttributesList(attributes)).toEqual({
        key: "stepEditor.duplicateAttributeName",
        vars: { name: "param" },
      });
    });

    it("validates default values for optional attributes", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "optional",
          name: "count",
          dataType: AttributeType.Number,
          defaultValue: "not-a-number",
        },
      ];

      const error = validateAttributesList(attributes);
      expect(error).toEqual({
        key: "stepEditor.invalidDefaultValue",
        vars: {
          name: "count",
          reason: "validation.jsonInvalid",
        },
      });
    });

    it("requires default values for const attributes", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "const",
          name: "flag",
          dataType: AttributeType.Boolean,
        },
      ];

      const error = validateAttributesList(attributes);
      expect(error).toEqual({
        key: "stepEditor.constDefaultRequired",
        vars: { name: "flag" },
      });
    });

    it("allows optional attributes without default values", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "optional",
          name: "maybe",
          dataType: AttributeType.String,
        },
      ];

      expect(validateAttributesList(attributes)).toBeNull();
    });
  });

  describe("createStepAttributes", () => {
    it("converts Attribute objects back to AttributeSpec", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "input_param",
          dataType: AttributeType.String,
        },
        {
          id: "attr-2",
          attrType: "optional",
          name: "optional_param",
          dataType: AttributeType.Number,
          defaultValue: "10",
          timeout: 3000,
        },
        {
          id: "attr-3",
          attrType: "const",
          name: "const_param",
          dataType: AttributeType.String,
          defaultValue: '"fixed"',
        },
        {
          id: "attr-4",
          attrType: "output",
          name: "output_result",
          dataType: AttributeType.String,
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.input_param.role).toBe(AttributeRole.Required);
      expect(result.optional_param.role).toBe(AttributeRole.Optional);
      expect(result.optional_param.default).toBe("10");
      expect(result.optional_param.timeout).toBe(3000);
      expect(result.const_param.role).toBe(AttributeRole.Const);
      expect(result.const_param.default).toBe('"fixed"');
      expect(result.output_result.role).toBe(AttributeRole.Output);
    });

    it("includes for_each when forEach is true", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "item",
          dataType: AttributeType.String,
          forEach: true,
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.item.for_each).toBe(true);
    });

    it("omits for_each when forEach is false or undefined", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "param",
          dataType: AttributeType.String,
          forEach: false,
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.param.for_each).toBeUndefined();
    });

    it("trims default values", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "optional",
          name: "value",
          dataType: AttributeType.String,
          defaultValue: "  trimmed  ",
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.value.default).toBe("trimmed");
    });

    it("omits default when it's only whitespace", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "optional",
          name: "value",
          dataType: AttributeType.String,
          defaultValue: "   ",
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.value.default).toBeUndefined();
    });

    it("adds mapping name and script when provided", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "input",
          name: "payload",
          dataType: AttributeType.Object,
          mappingName: "request",
          mappingLanguage: "jpath",
          mappingScript: "$.payload",
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.payload.mapping).toEqual({
        name: "request",
        script: {
          language: "jpath",
          script: "$.payload",
        },
      });
    });

    it("defaults mapping script language to lua", () => {
      const attributes: Attribute[] = [
        {
          id: "attr-1",
          attrType: "output",
          name: "result",
          dataType: AttributeType.Object,
          mappingScript: "$.result",
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.result.mapping?.script).toEqual({
        language: "lua",
        script: "$.result",
      });
    });
  });

  describe("getAttributeIconProps", () => {
    it("returns icon props for input attribute", () => {
      const props = getAttributeIconProps("input");
      expect(props).toBeDefined();
      expect(props.Icon).toBeDefined();
    });

    it("returns icon props for optional attribute", () => {
      const props = getAttributeIconProps("optional");
      expect(props).toBeDefined();
      expect(props.Icon).toBeDefined();
    });

    it("returns icon props for output attribute", () => {
      const props = getAttributeIconProps("output");
      expect(props).toBeDefined();
      expect(props.Icon).toBeDefined();
    });

    it("returns icon props for const attribute", () => {
      const props = getAttributeIconProps("const");
      expect(props).toBeDefined();
      expect(props.Icon).toBeDefined();
    });
  });

  describe("getValidationError", () => {
    it("returns error for missing step ID in create mode", () => {
      const error = getValidationError({
        isCreateMode: true,
        stepId: "  ",
        attributes: [],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.stepIdRequired" });
    });

    it("does not require step ID in edit mode", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "  ",
        attributes: [],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).not.toEqual({ key: "stepEditor.stepIdRequired" });
    });

    it("validates attributes using validateAttributesList", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [
          {
            id: "attr-1",
            attrType: "input",
            name: "  ",
            dataType: AttributeType.String,
          },
        ],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.attributeNameRequired" });
    });

    it("requires script content for script type", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "script",
        script: "  ",
        endpoint: "",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.scriptRequired" });
    });

    it("requires flow goals for flow type", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "flow",
        script: "",
        endpoint: "",
        httpTimeout: 0,
        flowGoals: "   ",
      });

      expect(error).toEqual({ key: "stepEditor.flowGoalsRequired" });
    });

    it("allows flow type without http or script config", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "flow",
        script: "",
        endpoint: "",
        httpTimeout: 0,
        flowGoals: "goal-a, goal-b",
      });

      expect(error).toBeNull();
    });

    it("requires endpoint for http type", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "sync",
        script: "",
        endpoint: "  ",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.endpointRequired" });
    });

    it("validates timeout is positive for http type", () => {
      let error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 0,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.timeoutPositive" });

      error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: -1000,
        flowGoals: "",
      });

      expect(error).toEqual({ key: "stepEditor.timeoutPositive" });
    });

    it("returns null for valid configuration", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [
          {
            id: "attr-1",
            attrType: "input",
            name: "param",
            dataType: AttributeType.String,
          },
        ],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toBeNull();
    });

    it("rejects duplicate mapping names for input attributes", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [
          {
            id: "attr-1",
            attrType: "input",
            name: "a",
            dataType: AttributeType.String,
            mappingName: "shared",
          },
          {
            id: "attr-2",
            attrType: "optional",
            name: "b",
            dataType: AttributeType.String,
            mappingName: "shared",
          },
        ],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({
        key: "stepEditor.duplicateMappingName",
        vars: { name: "shared" },
      });
    });

    it("rejects const attribute mappings", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [
          {
            id: "attr-1",
            attrType: "const",
            name: "const_value",
            dataType: AttributeType.String,
            defaultValue: '"x"',
            mappingName: "illegal",
          },
        ],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({
        key: "stepEditor.constMappingNotAllowed",
        vars: { name: "const_value" },
      });
    });

    it("requires mapping language when mapping script is set", () => {
      const error = getValidationError({
        isCreateMode: false,
        stepId: "step-1",
        attributes: [
          {
            id: "attr-1",
            attrType: "output",
            name: "result",
            dataType: AttributeType.String,
            mappingScript: "$.result",
            mappingLanguage: " ",
          },
        ],
        stepType: "sync",
        script: "",
        endpoint: "https://example.com",
        httpTimeout: 5000,
        flowGoals: "",
      });

      expect(error).toEqual({
        key: "stepEditor.mappingLanguageRequired",
        vars: { name: "result" },
      });
    });
  });
});
