import { act, renderHook } from "@testing-library/react";
import { AttributeType, SCRIPT_LANGUAGE_LUA, Step } from "@/app/api";
import { useStepEditorForm } from "./useStepEditorForm";
import { t } from "@/app/testUtils/i18n";

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
  type: "service",
  attributes: {},
  http: { invoke: { endpoint: "https://example.com", timeout: 1000 } },
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
      useStepEditorForm({ step: null, onUpdate, onClose })
    );

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe(t("stepEditor.stepIdRequired"));
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("creates http step when valid", async () => {
    const createdStep = buildStep({ name: "Created" });
    registerStep.mockResolvedValue(createdStep);

    const { result } = renderHook(() =>
      useStepEditorForm({ step: null, onUpdate, onClose })
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
      useStepEditorForm({ step: null, onUpdate, onClose })
    );

    act(() => {
      result.current.setStepType("script");
      result.current.setStepId("script-step");
      result.current.setScriptLanguage(SCRIPT_LANGUAGE_LUA);
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe(t("stepEditor.scriptRequired"));
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("requires flow goals for flow type", async () => {
    const { result } = renderHook(() =>
      useStepEditorForm({ step: null, onUpdate, onClose })
    );

    act(() => {
      result.current.setStepType("flow");
      result.current.setStepId("flow-step");
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.error).toBe(t("stepEditor.flowGoalsRequired"));
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
      },
    });
    registerStep.mockResolvedValue(createdStep);

    const { result } = renderHook(() =>
      useStepEditorForm({ step: null, onUpdate, onClose })
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
      result.current.updateAttribute(inputAttrId, "role", "required");
      result.current.updateAttribute(inputAttrId, "mappingName", "child_input");
      result.current.updateAttribute(outputAttrId, "name", "output");
      result.current.updateAttribute(outputAttrId, "role", "output");
      result.current.updateAttribute(
        outputAttrId,
        "mappingName",
        "child_output"
      );
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(registerStep).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "flow",
        flow: {
          goals: ["goal-1", "goal-2"],
          compensate: false,
        },
        attributes: expect.objectContaining({
          input: expect.objectContaining({
            required: expect.objectContaining({
              mapping: expect.objectContaining({ name: "child_input" }),
            }),
          }),
          output: expect.objectContaining({
            output: expect.objectContaining({
              mapping: expect.objectContaining({ name: "child_output" }),
            }),
          }),
        }),
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
      useStepEditorForm({ step: null, onUpdate, onClose })
    );

    act(() => {
      result.current.setStepId("with-attrs");
      result.current.setEndpoint("https://api.example.com");
      result.current.addAttribute();
    });

    const attrId = result.current.attributes[0].id;

    act(() => {
      result.current.updateAttribute(attrId, "name", "value");
      result.current.updateAttribute(attrId, "role", "optional");
      result.current.updateAttribute(attrId, "dataType", AttributeType.Number);
      result.current.updateAttribute(attrId, "defaultValue", '"abc"');
    });

    await act(async () => {
      await result.current.handleSave();
    });

    const expectedError = t("stepEditor.invalidDefaultValue", {
      name: "value",
      reason: t("validation.jsonNumber"),
    });
    expect(result.current.error).toBe(expectedError);
    expect(registerStep).not.toHaveBeenCalled();
  });

  it("updates an existing step", async () => {
    const existingStep = buildStep({
      id: "existing-step",
      http: { invoke: { endpoint: "https://example.com", timeout: 1500 } },
    });
    const updatedStep = buildStep({
      id: "existing-step",
      name: "Updated",
    });

    updateStep.mockResolvedValue(updatedStep);

    const { result } = renderHook(() =>
      useStepEditorForm({ step: existingStep, onUpdate, onClose })
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

  it("serializes compensation attribute selections", () => {
    const existingStep = buildStep({
      handling: "compensated",
      attributes: {
        request: {
          role: "required",
          type: AttributeType.String,
          compensated: true,
        },
        result: {
          role: "output",
          type: AttributeType.String,
          compensated: true,
        },
      },
      http: {
        invoke: { endpoint: "https://example.com" },
        compensate: { endpoint: "https://example.com/undo" },
      },
    });
    const { result } = renderHook(() =>
      useStepEditorForm({ step: existingStep, onUpdate, onClose })
    );

    const serialized = JSON.parse(result.current.getSerializedStepData());

    expect(serialized.handling).toBe("compensated");
    expect(serialized.attributes.request.compensated).toBe(true);
    expect(serialized.attributes.result.compensated).toBe(true);
    expect(serialized.http.compensate).toEqual({
      endpoint: "https://example.com/undo",
      method: "POST",
      mode: "sync",
    });
  });

  it("clears compensated attributes when handling changes", () => {
    const existingStep = buildStep({
      handling: "compensated",
      attributes: {
        request: {
          role: "required",
          type: AttributeType.String,
          compensated: true,
        },
      },
      http: {
        invoke: { endpoint: "https://example.com" },
        compensate: { endpoint: "https://example.com/undo" },
      },
    });
    const { result } = renderHook(() =>
      useStepEditorForm({ step: existingStep, onUpdate, onClose })
    );

    act(() => result.current.setHandling("standard"));

    expect(result.current.attributes[0].compensated).toBeUndefined();
  });

  it("clears handling for non-HTTP types", () => {
    const existingStep = buildStep({
      handling: "compensated",
      attributes: {
        request: {
          role: "required",
          type: AttributeType.String,
          compensated: true,
        },
      },
      http: {
        invoke: { endpoint: "https://example.com" },
        compensate: { endpoint: "https://example.com/undo" },
      },
    });
    const { result } = renderHook(() =>
      useStepEditorForm({ step: existingStep, onUpdate, onClose })
    );

    act(() => result.current.setStepType("flow"));

    expect(result.current.handling).toBe("standard");
    expect(result.current.attributes[0].compensated).toBeUndefined();
  });

  describe("attribute type side effects", () => {
    it("clears match script for non-input roles", () => {
      const { result } = renderHook(() =>
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "role", "required");
        result.current.updateAttribute(attrId, "matchScript", "$.foo");
      });

      act(() => {
        result.current.updateAttribute(attrId, "role", "optional");
      });

      expect(result.current.attributes[0].matchScript).toBeUndefined();
    });

    it("clears collect and forEach when changing to output", () => {
      const { result } = renderHook(() =>
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "collect", "some");
        result.current.updateAttribute(attrId, "forEach", true);
        result.current.updateAttribute(attrId, "role", "output");
      });

      expect(result.current.attributes[0].collect).toBe("first");
      expect(result.current.attributes[0].forEach).toBe(false);
    });

    it("clears collect and forEach when changing to meta", () => {
      const { result } = renderHook(() =>
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "collect", "all");
        result.current.updateAttribute(attrId, "forEach", true);
        result.current.updateAttribute(attrId, "role", "meta");
      });

      expect(result.current.attributes[0].collect).toBe("first");
      expect(result.current.attributes[0].forEach).toBe(false);
    });
  });

  describe("validation error clearing", () => {
    it("clears errors for non-optional roles", () => {
      const { result } = renderHook(() =>
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "test");
        result.current.updateAttribute(attrId, "role", "optional");
        result.current.updateAttribute(
          attrId,
          "dataType",
          AttributeType.Number
        );
        result.current.updateAttribute(attrId, "defaultValue", "invalid");
      });

      expect(result.current.attributes[0].validationError).toBeDefined();

      act(() => {
        result.current.updateAttribute(attrId, "role", "required");
      });

      expect(result.current.attributes[0].validationError).toBeUndefined();
    });

    it("validates default value when attrType is optional", () => {
      const { result } = renderHook(() =>
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "count");
        result.current.updateAttribute(attrId, "role", "optional");
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
        useStepEditorForm({ step: null, onUpdate, onClose })
      );

      act(() => {
        result.current.setStepId("step-1");
        result.current.setEndpoint("https://api.example.com");
        result.current.addAttribute();
      });

      const attrId = result.current.attributes[0].id;

      act(() => {
        result.current.updateAttribute(attrId, "name", "count");
        result.current.updateAttribute(attrId, "role", "optional");
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
        useStepEditorForm({ step: null, onUpdate, onClose })
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
        useStepEditorForm({ step: null, onUpdate, onClose })
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
