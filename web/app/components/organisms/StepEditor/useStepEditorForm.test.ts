import { act, renderHook } from "@testing-library/react";
import { AttributeType, SCRIPT_LANGUAGE_LUA, Step } from "@/app/api";
import { useStepEditorForm } from "./useStepEditorForm";

const registerStep = jest.fn();
const updateStep = jest.fn();

jest.mock("@/app/api", () => {
  const actual = jest.requireActual("@/app/api");
  return {
    ...actual,
    ArgyllApi: class MockArgyllApi {
      registerStep = registerStep;
      updateStep = updateStep;
    },
  };
});

const buildStep = (overrides: Partial<Step> = {}): Step => ({
  id: "step-1",
  name: "Step 1",
  type: "sync",
  attributes: {},
  http: { endpoint: "https://example.com", timeout: 1000 },
  ...overrides,
});

describe("useStepEditorForm", () => {
  const onUpdate = jest.fn();
  const onClose = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("validates required fields in create mode", async () => {
    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe("Step ID is required");
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("creates http step when valid", async () => {
    const createdStep = buildStep({ name: "Created" });
    registerStep.mockResolvedValue(createdStep);

    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    act(() => {
      result.current.setStepId("new-step");
      result.current.setEndpoint("https://api.example.com");
      result.current.setName("Created");
      result.current.setHttpTimeout(2000);
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(registerStep).toHaveBeenCalledTimes(1);
    expect(onUpdate).toHaveBeenCalledWith(createdStep);
    expect(onClose).toHaveBeenCalled();
    expect(result.current.error).toBeNull();
  });

  it("requires script content for script type", async () => {
    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    act(() => {
      result.current.setStepType("script");
      result.current.setStepId("script-step");
      result.current.setScriptLanguage(SCRIPT_LANGUAGE_LUA);
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe("Script code is required");
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("requires flow goals for flow type", async () => {
    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    act(() => {
      result.current.setStepType("flow");
      result.current.setStepId("flow-step");
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe("Flow goals are required");
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("creates flow step when valid", async () => {
    const createdStep = buildStep({
      id: "flow-step",
      name: "Flow Step",
      type: "flow",
      http: undefined,
      script: undefined,
      flow: {
        goals: ["goal-1", "goal-2"],
        input_map: { input: "child_input" },
        output_map: { child_output: "output" },
      },
    });
    registerStep.mockResolvedValue(createdStep);

    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    act(() => {
      result.current.setStepId("flow-step");
      result.current.setName("Flow Step");
      result.current.setStepType("flow");
      result.current.setFlowGoals("goal-1, goal-2");
      result.current.addAttribute();
      result.current.addAttribute();
    });

    const inputAttrId = result.current.attributes[0].id;
    const outputAttrId = result.current.attributes[1].id;

    act(() => {
      result.current.updateAttribute(inputAttrId, "name", "input");
      result.current.updateAttribute(inputAttrId, "attrType", "input");
      result.current.updateAttribute(inputAttrId, "flowMap", "child_input");
      result.current.updateAttribute(outputAttrId, "name", "output");
      result.current.updateAttribute(outputAttrId, "attrType", "output");
      result.current.updateAttribute(outputAttrId, "flowMap", "child_output");
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(registerStep).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "flow",
        flow: {
          goals: ["goal-1", "goal-2"],
          input_map: { input: "child_input" },
          output_map: { child_output: "output" },
        },
        http: undefined,
        script: undefined,
      })
    );
    expect(onUpdate).toHaveBeenCalledWith(createdStep);
    expect(onClose).toHaveBeenCalled();
    expect(result.current.error).toBeNull();
  });

  it("reports invalid attribute defaults", async () => {
    const { result } = renderHook(() =>
      useStepEditorForm(null, onUpdate, onClose)
    );

    act(() => {
      result.current.setStepId("with-attrs");
      result.current.setEndpoint("https://api.example.com");
      result.current.addAttribute();
    });

    const attrId = result.current.attributes[0].id;

    act(() => {
      result.current.updateAttribute(attrId, "name", "value");
      result.current.updateAttribute(attrId, "attrType", "optional");
      result.current.updateAttribute(attrId, "dataType", AttributeType.Number);
      result.current.updateAttribute(attrId, "defaultValue", '"abc"');
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe(
      'Invalid default value for "value": Must be a valid number'
    );
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("updates an existing step", async () => {
    const existingStep = buildStep({
      id: "existing-step",
      http: { endpoint: "https://example.com", timeout: 1500 },
    });
    const updatedStep = buildStep({
      id: "existing-step",
      name: "Updated",
    });

    updateStep.mockResolvedValue(updatedStep);

    const { result } = renderHook(() =>
      useStepEditorForm(existingStep, onUpdate, onClose)
    );

    act(() => {
      result.current.setName("Updated");
      result.current.setHttpTimeout(2500);
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(updateStep).toHaveBeenCalledWith("existing-step", expect.anything());
    expect(onUpdate).toHaveBeenCalledWith(updatedStep);
    expect(onClose).toHaveBeenCalled();
  });

  describe("attribute type cycling", () => {
    it("cycles attribute type from input to optional", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "attrType", "input");
      });

      expect(result.current.attributes[0].attrType).toBe("input");

      act(() => {
        result.current.cycleAttributeType(attrId, "input");
      });

      expect(result.current.attributes[0].attrType).toBe("optional");
    });

    it("cycles attribute type from optional to const", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "attrType", "optional");
      });

      act(() => {
        result.current.cycleAttributeType(attrId, "optional");
      });

      expect(result.current.attributes[0].attrType).toBe("const");
    });

    it("cycles attribute type from const to output", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "attrType", "const");
      });

      act(() => {
        result.current.cycleAttributeType(attrId, "const");
      });

      expect(result.current.attributes[0].attrType).toBe("output");
    });

    it("cycles attribute type from output back to input", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "attrType", "output");
      });

      act(() => {
        result.current.cycleAttributeType(attrId, "output");
      });

      expect(result.current.attributes[0].attrType).toBe("input");
    });
  });

  describe("validation error clearing", () => {
    it("clears validation error when changing to non-optional type", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "attrType", "optional");
        result.current.updateAttribute(
          attrId,
          "dataType",
          AttributeType.Number
        );
        result.current.updateAttribute(attrId, "defaultValue", "invalid");
      });

      expect(result.current.attributes[0].validationError).toBeDefined();

      act(() => {
        result.current.updateAttribute(attrId, "attrType", "input");
      });

      expect(result.current.attributes[0].validationError).toBeUndefined();
    });

    it("validates default value when attrType is optional", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "count");
        result.current.updateAttribute(attrId, "attrType", "optional");
        result.current.updateAttribute(
          attrId,
          "dataType",
          AttributeType.Number
        );
        result.current.updateAttribute(
          attrId,
          "defaultValue",
          '"not-a-number"'
        );
      });

      expect(result.current.attributes[0].validationError).toBeDefined();
    });

    it("accepts valid default value for optional attribute", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "count");
        result.current.updateAttribute(attrId, "attrType", "optional");
        result.current.updateAttribute(
          attrId,
          "dataType",
          AttributeType.Number
        );
        result.current.updateAttribute(attrId, "defaultValue", "42");
      });

      expect(result.current.attributes[0].validationError).toBeUndefined();
    });
  });

  describe("attribute removal", () => {
    it("removes attribute by id", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      expect(result.current.attributes).toHaveLength(1);

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.removeAttribute(attrId);
      });

      expect(result.current.attributes).toHaveLength(0);
    });

    it("removes correct attribute when multiple exist", () => {
      const { result } = renderHook(() =>
        useStepEditorForm(null, onUpdate, onClose)
      );

      const attrIds: string[] = [];

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
        result.current.addAttribute();
        result.current.addAttribute();
      });

      expect(result.current.attributes).toHaveLength(3);
      attrIds.push(result.current.attributes[0].id);
      attrIds.push(result.current.attributes[1].id);
      attrIds.push(result.current.attributes[2].id);

      act(() => {
        result.current.removeAttribute(attrIds[1]);
      });

      expect(result.current.attributes).toHaveLength(2);
      expect(result.current.attributes[0].id).toBe(attrIds[0]);
      expect(result.current.attributes[1].id).toBe(attrIds[2]);
    });
  });
});
