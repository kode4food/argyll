import {
  generatePadded,
  generateWorkflowId,
  sanitizeWorkflowID,
} from "./workflowUtils";

describe("workflowUtils", () => {
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

  describe("generateWorkflowId", () => {
    test("generates ID with workflow prefix", () => {
      const result = generateWorkflowId();
      expect(result).toMatch(/^workflow-\d{8}$/);
    });

    test("generates unique IDs on consecutive calls", () => {
      const ids = new Set();
      for (let i = 0; i < 10; i++) {
        ids.add(generateWorkflowId());
      }
      expect(ids.size).toBeGreaterThan(1);
    });

    test("includes padded numeric portion", () => {
      const result = generateWorkflowId();
      const parts = result.split("-");
      expect(parts).toHaveLength(2);
      expect(parts[0]).toBe("workflow");
      expect(parts[1]).toMatch(/^\d{8}$/);
    });
  });

  describe("sanitizeWorkflowID", () => {
    test("converts to lowercase", () => {
      expect(sanitizeWorkflowID("WORKFLOW-ABC")).toBe("workflow-abc");
    });

    test("replaces spaces with hyphens", () => {
      expect(sanitizeWorkflowID("my workflow id")).toBe("my-workflow-id");
    });

    test("removes special characters", () => {
      expect(sanitizeWorkflowID("workflow@#$%id")).toBe("workflowid");
    });

    test("preserves valid characters", () => {
      expect(sanitizeWorkflowID("workflow-123_test.v2+1")).toBe(
        "workflow-123_test.v2+1"
      );
    });

    test("removes leading hyphens", () => {
      expect(sanitizeWorkflowID("---workflow")).toBe("workflow");
    });

    test("removes trailing hyphens", () => {
      expect(sanitizeWorkflowID("workflow---")).toBe("workflow");
    });

    test("removes both leading and trailing hyphens", () => {
      expect(sanitizeWorkflowID("---workflow---")).toBe("workflow");
    });

    test("handles empty string", () => {
      expect(sanitizeWorkflowID("")).toBe("");
    });

    test("handles string with only invalid characters", () => {
      expect(sanitizeWorkflowID("@#$%^&*()")).toBe("");
    });

    test("handles complex mixed input", () => {
      expect(sanitizeWorkflowID("  My Workflow!! ID@123  ")).toBe(
        "my-workflow-id123"
      );
    });

    test("preserves dots and underscores", () => {
      expect(sanitizeWorkflowID("workflow.v1_test")).toBe("workflow.v1_test");
    });

    test("preserves plus signs", () => {
      expect(sanitizeWorkflowID("workflow+v2")).toBe("workflow+v2");
    });

    test("handles consecutive spaces", () => {
      expect(sanitizeWorkflowID("my   workflow   id")).toBe(
        "my---workflow---id"
      );
    });

    test("handles unicode characters", () => {
      expect(sanitizeWorkflowID("workflow-日本語")).toBe("workflow");
    });

    test("handles numbers", () => {
      expect(sanitizeWorkflowID("workflow-12345")).toBe("workflow-12345");
    });
  });
});
