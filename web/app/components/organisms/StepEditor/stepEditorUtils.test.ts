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
            description: "",
          },
          optional_arg: {
            role: AttributeRole.Optional,
            type: AttributeType.Number,
            default: 42,
            description: "",
          },
          output_arg: {
            role: AttributeRole.Output,
            type: AttributeType.String,
            description: "",
          },
        },
      };

      const result = buildAttributesFromStep(step);

      expect(result).toHaveLength(3);

      const inputAttrs = result.filter((a) => a.attrType === "input");
      const optionalAttrs = result.filter((a) => a.attrType === "optional");
      const outputAttrs = result.filter((a) => a.attrType === "output");

      expect(inputAttrs).toHaveLength(1);
      expect(inputAttrs[0].name).toBe("required_arg");

      expect(optionalAttrs).toHaveLength(1);
      expect(optionalAttrs[0].name).toBe("optional_arg");
      expect(optionalAttrs[0].defaultValue).toBe("42");

      expect(outputAttrs).toHaveLength(1);
      expect(outputAttrs[0].name).toBe("output_arg");
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
        },
        {
          id: "attr-3",
          attrType: "output",
          name: "output_result",
          dataType: AttributeType.String,
        },
      ];

      const result = createStepAttributes(attributes);

      expect(result.input_param.role).toBe(AttributeRole.Required);
      expect(result.optional_param.role).toBe(AttributeRole.Optional);
      expect(result.optional_param.default).toBe("10");
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
      });

      expect(error).toEqual({ key: "stepEditor.scriptRequired" });
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
      });

      expect(error).toBeNull();
    });
  });
});
