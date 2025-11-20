import { generatePadded, generateFlowId, sanitizeFlowID } from "./flowUtils";

describe("flowUtils", () => {
  describe("generatePadded", () => {
    test("generates a padded numeric string", () => {
      const result = generatePadded();
      expect(result).toMatch(/^\d{8}$/);
    });

    test("generates different values on consecutive calls", () => {
      const results = new Set();
      for (let i = 0; i < 10; i++) {
        results.add(generatePadded());
      }
      expect(results.size).toBeGreaterThan(1);
    });

    test("pads short numbers with leading zeros", () => {
      jest.spyOn(Math, "random").mockReturnValue(0.00000001);
      const result = generatePadded();
      expect(result).toBe("00000001");
      jest.spyOn(Math, "random").mockRestore();
    });

    test("handles larger numbers within range", () => {
      jest.spyOn(Math, "random").mockReturnValue(0.99999999);
      const result = generatePadded();
      expect(result).toHaveLength(8);
      expect(Number(result)).toBeLessThan(100000000);
      jest.spyOn(Math, "random").mockRestore();
    });
  });

  describe("generateFlowId", () => {
    test("generates ID with flow prefix", () => {
      const result = generateFlowId();
      expect(result).toMatch(/^flow-\d{8}$/);
    });

    test("generates unique IDs on consecutive calls", () => {
      const ids = new Set();
      for (let i = 0; i < 10; i++) {
        ids.add(generateFlowId());
      }
      expect(ids.size).toBeGreaterThan(1);
    });

    test("includes padded numeric portion", () => {
      const result = generateFlowId();
      const parts = result.split("-");
      expect(parts).toHaveLength(2);
      expect(parts[0]).toBe("flow");
      expect(parts[1]).toMatch(/^\d{8}$/);
    });
  });

  describe("sanitizeFlowID", () => {
    test("converts to lowercase", () => {
      expect(sanitizeFlowID("FLOW-ABC")).toBe("flow-abc");
    });

    test("replaces spaces with hyphens", () => {
      expect(sanitizeFlowID("my flow id")).toBe("my-flow-id");
    });

    test("removes special characters", () => {
      expect(sanitizeFlowID("flow@#$%id")).toBe("flowid");
    });

    test("preserves valid characters", () => {
      expect(sanitizeFlowID("flow-123_test.v2+1")).toBe("flow-123_test.v2+1");
    });

    test("removes leading hyphens", () => {
      expect(sanitizeFlowID("---flow")).toBe("flow");
    });

    test("removes trailing hyphens", () => {
      expect(sanitizeFlowID("flow---")).toBe("flow");
    });

    test("removes both leading and trailing hyphens", () => {
      expect(sanitizeFlowID("---flow---")).toBe("flow");
    });

    test("handles empty string", () => {
      expect(sanitizeFlowID("")).toBe("");
    });

    test("handles string with only invalid characters", () => {
      expect(sanitizeFlowID("@#$%^&*()")).toBe("");
    });

    test("handles complex mixed input", () => {
      expect(sanitizeFlowID("  My Flow!! ID@123  ")).toBe("my-flow-id123");
    });

    test("preserves dots and underscores", () => {
      expect(sanitizeFlowID("flow.v1_test")).toBe("flow.v1_test");
    });

    test("preserves plus signs", () => {
      expect(sanitizeFlowID("flow+v2")).toBe("flow+v2");
    });

    test("handles consecutive spaces", () => {
      expect(sanitizeFlowID("my   flow   id")).toBe("my---flow---id");
    });

    test("handles unicode characters", () => {
      expect(sanitizeFlowID("flow-日本語")).toBe("flow");
    });

    test("handles numbers", () => {
      expect(sanitizeFlowID("flow-12345")).toBe("flow-12345");
    });
  });
});
