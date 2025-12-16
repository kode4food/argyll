import { act, renderHook } from "@testing-library/react";
import { AttributeType, SCRIPT_LANGUAGE_LUA, Step } from "@/app/api";
import { useStepEditorForm } from "./useStepEditorForm";

const registerStep = jest.fn();
const updateStep = jest.fn();

jest.mock("@/app/api", () => {
  const actual = jest.requireActual("@/app/api");
  return {
    ...actual,
    ArgyllApi: jest.fn().mockImplementation(() => ({
      registerStep,
      updateStep,
    })),
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
});
