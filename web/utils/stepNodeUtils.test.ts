import {
  groupAttributesByRole,
  generateHandleId,
  buildProvenanceMap,
} from "./stepNodeUtils";
import { AttributeRole, AttributeType } from "@/app/api";

describe("stepNodeUtils", () => {
  describe("groupAttributesByRole", () => {
    it("groups attributes by their role", () => {
      const attributes = {
        input1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        input2: {
          role: AttributeRole.Optional,
          type: AttributeType.String,
          description: "",
        },
        output1: {
          role: AttributeRole.Output,
          type: AttributeType.String,
          description: "",
        },
      };

      const result = groupAttributesByRole(attributes);

      expect(result.required).toContain("input1");
      expect(result.optional).toContain("input2");
      expect(result.output).toContain("output1");
    });

    it("sorts attribute names alphabetically", () => {
      const attributes = {
        zebra: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        apple: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        banana: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      };

      const result = groupAttributesByRole(attributes);

      expect(result.required).toEqual(["apple", "banana", "zebra"]);
    });

    it("returns empty arrays for missing roles", () => {
      const attributes = {
        input1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      };

      const result = groupAttributesByRole(attributes);

      expect(result.required).toHaveLength(1);
      expect(result.optional).toHaveLength(0);
      expect(result.output).toHaveLength(0);
    });

    it("handles empty attributes object", () => {
      const result = groupAttributesByRole({});

      expect(result.required).toEqual([]);
      expect(result.optional).toEqual([]);
      expect(result.output).toEqual([]);
    });
  });

  describe("generateHandleId", () => {
    it("generates input handle ID with type prefix", () => {
      const id = generateHandleId("required", "username");
      expect(id).toBe("input-required-username");
    });

    it("generates optional input handle ID", () => {
      const id = generateHandleId("optional", "apiKey");
      expect(id).toBe("input-optional-apiKey");
    });

    it("generates output handle ID without type prefix", () => {
      const id = generateHandleId("output", "result");
      expect(id).toBe("output-result");
    });

    it("preserves special characters in names", () => {
      const id = generateHandleId("required", "user-id");
      expect(id).toBe("input-required-user-id");
    });
  });

  describe("buildProvenanceMap", () => {
    it("builds map from flow state", () => {
      const flowState = {
        attr1: [{ step: "step-1" }],
        attr2: [{ step: "step-2" }],
        attr3: [{ step: "step-1" }],
      };

      const map = buildProvenanceMap(flowState);

      expect(map.get("attr1")).toBe("step-1");
      expect(map.get("attr2")).toBe("step-2");
      expect(map.get("attr3")).toBe("step-1");
    });

    it("ignores attributes without step", () => {
      const flowState = {
        attr1: [{ step: "step-1" }],
        attr2: [{ other: "value" }],
        attr3: [],
      };

      const map = buildProvenanceMap(flowState);

      expect(map.has("attr1")).toBe(true);
      expect(map.has("attr2")).toBe(false);
      expect(map.has("attr3")).toBe(false);
    });

    it("uses the first value when multiple values are present", () => {
      const flowState = {
        attr1: [{ step: "step-1" }, { step: "step-2" }],
      };

      const map = buildProvenanceMap(flowState);

      expect(map.get("attr1")).toBe("step-1");
    });

    it("handles empty flow state", () => {
      const map = buildProvenanceMap({});
      expect(map.size).toBe(0);
    });

    it("handles undefined flow state", () => {
      const map = buildProvenanceMap(undefined);
      expect(map.size).toBe(0);
    });

    it("returns a Map object", () => {
      const map = buildProvenanceMap({ attr1: [{ step: "step-1" }] });
      expect(map).toBeInstanceOf(Map);
    });
  });
});
