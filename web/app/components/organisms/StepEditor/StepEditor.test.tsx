import React from "react";
import StepEditor from "./StepEditor";
import { t } from "@/app/testUtils/i18n";
import { ArgyllApi, AttributeRole, AttributeType } from "@/app/api";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { Space, Step } from "@/app/api";

jest.requireActual("@/app/api");

let stepsInStore: Step[] = [];
let spacesInStore: Space[] = [];
let selectedSpaceId: string | null = null;

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({ spaceId: selectedSpaceId }),
}));

jest.mock("@/app/store/flowStore", () => ({
  useSteps: () => stepsInStore,
  useSpaces: () => spacesInStore,
  useSpaceSelection: () => ({}),
}));

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  ArgyllApi: jest.fn(),
  api: {
    getExecutionPlan: jest.fn().mockResolvedValue({
      steps: {},
      required: [],
      attributes: {},
      goals: [],
    }),
  },
}));

jest.mock("@/app/components/molecules/ScriptEditor", () => ({
  __esModule: true,
  default: ({ value, onChange }: any) => (
    <textarea
      data-testid="script-editor"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock("@/app/components/molecules/DurationInput", () => ({
  __esModule: true,
  default: ({ value, onChange }: any) => (
    <input
      data-testid="duration-input"
      type="text"
      value={value || ""}
      onChange={(e) => {
        // Simulate simple parsing for test
        const val = e.target.value;
        if (!val) {
          onChange(0);
        } else if (/^\d+$/.test(val)) {
          onChange(parseInt(val));
        } else {
          onChange(parseInt(val) || 5000);
        }
      }}
    />
  ),
}));

const MockedArgyllApi = ArgyllApi as jest.MockedClass<typeof ArgyllApi>;

describe("StepEditor", () => {
  const createHttpStep = (): Step => ({
    id: "step-1",
    name: "Test HTTP Step",
    type: "service",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      input2: { role: AttributeRole.Optional, type: AttributeType.Number },
      result: { role: AttributeRole.Output, type: AttributeType.String },
    },
    http: {
      invoke: { endpoint: "http://localhost:8080/test", timeout: 5000 },
      health: "http://localhost:8080/health",
    },
    predicate: {
      language: "lua",
      script: "return temperature > 100",
    },
  });

  const createScriptStep = (): Step => ({
    id: "step-2",
    name: "Test Script Step",
    type: "script",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      result: { role: AttributeRole.Output, type: AttributeType.String },
    },
    script: {
      language: "lua",
      script: "return {result = 42}",
    },
    predicate: {
      language: "lua",
      script: "return value > 10",
    },
  });

  const mockOnClose = jest.fn();
  const mockOnUpdate = jest.fn();
  const mockUpdateStep = jest.fn();

  beforeEach(() => {
    stepsInStore = [];
    spacesInStore = [];
    selectedSpaceId = null;
    MockedArgyllApi.mockImplementation(
      () =>
        ({
          updateStep: mockUpdateStep,
        }) as Partial<ArgyllApi> as ArgyllApi
    );

    document.body.innerHTML = "";
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  const createFlowStep = (): Step => ({
    id: "step-flow",
    name: "Test Flow Step",
    type: "flow",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      output1: { role: AttributeRole.Output, type: AttributeType.String },
    },
    flow: {
      goals: [],
    },
  });

  const createConstStep = (): Step => ({
    id: "step-const",
    name: "Const Step",
    type: "service",
    attributes: {
      color: {
        role: AttributeRole.Const,
        type: AttributeType.String,
        const: { value: "blue" },
      },
    },
    http: {
      invoke: { endpoint: "http://localhost:8080/test", timeout: 5000 },
    },
  });

  test("renders modal with HTTP step data", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.modalEditTitle", { id: "step-1" }))
      ).toBeInTheDocument();
      expect(screen.getByDisplayValue("Test HTTP Step")).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("http://localhost:8080/test")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("http://localhost:8080/health")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("return temperature > 100")
      ).toBeInTheDocument();
    });
  });

  test("renders modal with script step data", async () => {
    const step = createScriptStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.modalEditTitle", { id: "step-2" }))
      ).toBeInTheDocument();
      expect(screen.getByDisplayValue("Test Script Step")).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("return {result = 42}")
      ).toBeInTheDocument();
      expect(screen.getByDisplayValue("return value > 10")).toBeInTheDocument();
    });
  });

  test("renders required args", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("input1")).toBeInTheDocument();
    });
  });

  test("edits required match", async () => {
    const step = createHttpStep();
    step.attributes.input1.required = {
      match: {
        language: "jpath",
        script: "$.kind",
      },
    };
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    const matchInput = await screen.findByDisplayValue("$.kind");
    const matchLangBtn = screen.getByLabelText(
      t("stepEditor.matchLanguageLabel")
    );
    fireEvent.click(matchLangBtn);
    fireEvent.click(screen.getByRole("option", { name: "Lua" }));
    fireEvent.change(matchInput, { target: { value: "$.product_type" } });

    fireEvent.click(screen.getByText(t("stepEditor.save")));

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          attributes: expect.objectContaining({
            input1: expect.objectContaining({
              required: expect.objectContaining({
                match: {
                  language: "lua",
                  script: "$.product_type",
                },
              }),
            }),
          }),
        })
      );
    });
  });

  test("uses match placeholders", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    expect(
      await screen.findByPlaceholderText(
        t("stepEditor.matchScriptPlaceholderJPath")
      )
    ).toBeInTheDocument();

    const matchLanguageSelect = screen.getByLabelText(
      t("stepEditor.matchLanguageLabel")
    );

    fireEvent.click(matchLanguageSelect);
    fireEvent.click(screen.getByRole("option", { name: "Lua" }));
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText(t("stepEditor.matchScriptPlaceholderLua"))
      ).toBeInTheDocument();
    });
  });

  test("renders optional args with deadline", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("input2")).toBeInTheDocument();
      expect(
        screen.getAllByTestId("duration-input").length
      ).toBeGreaterThanOrEqual(1);
    });
  });

  test("renders output args", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("result")).toBeInTheDocument();
    });
  });

  test("renders const default value input", async () => {
    const step = createConstStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("blue")).toBeInTheDocument();
    });
  });

  test("shows placeholder row when no attributes exist", async () => {
    render(
      <StepEditor step={null} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.modalCreateTitle"))
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          /Attributes describe how steps share data with each other/i
        )
      ).toBeInTheDocument();
    });
  });

  test("seeds labels from the selected Space", async () => {
    spacesInStore = [
      {
        id: "risk",
        name: "Risk",
        selector: { domain: "risk" },
      },
    ];
    selectedSpaceId = "risk";

    render(
      <StepEditor step={null} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("domain")).toBeInTheDocument();
      expect(screen.getByDisplayValue("risk")).toBeInTheDocument();
    });
  });

  test("seeds no labels without a selected Space", async () => {
    render(
      <StepEditor step={null} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.modalCreateTitle"))
      ).toBeInTheDocument();
    });
    expect(
      screen.queryByPlaceholderText(t("stepEditor.labelKeyPlaceholder"))
    ).not.toBeInTheDocument();
  });

  test("updates step name", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const nameInput = screen.getByDisplayValue("Test HTTP Step");
      fireEvent.change(nameInput, { target: { value: "New Name" } });
      expect(screen.getByDisplayValue("New Name")).toBeInTheDocument();
    });
  });

  test("renders flow mapping dropdown options", async () => {
    const step = createFlowStep();
    stepsInStore = [
      {
        id: "child-step",
        name: "Child Step",
        type: "service",
        attributes: {},
        http: {
          invoke: { endpoint: "http://localhost:8080/child", timeout: 5000 },
        },
      },
    ];

    const { api } = jest.requireMock("@/app/api");
    const plan = {
      steps: {
        "child-step": {
          id: "child-step",
          name: "Child Step",
          type: "service",
          attributes: {
            in1: { role: AttributeRole.Required, type: AttributeType.String },
            out1: { role: AttributeRole.Output, type: AttributeType.String },
          },
        },
      },
      required: [],
      attributes: {},
      goals: ["child-step"],
    };
    api.getExecutionPlan.mockResolvedValue(plan);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    fireEvent.click(
      await screen.findByRole("button", { name: t("stepEditor.typeFlowLabel") })
    );
    await waitFor(() => {
      const flowButton = screen.getByTitle(t("stepEditor.typeFlowTitle"));
      fireEvent.click(flowButton);
    });

    await waitFor(() => {
      const goalChip = screen.getByText("child-step");
      fireEvent.click(goalChip);
    });

    const expandInputMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} input1`,
    });
    fireEvent.click(expandInputMappingButton);

    fireEvent.click(await screen.findByRole("button", { name: "input1" }));
    expect(
      await screen.findByRole("option", { name: "in1" })
    ).toBeInTheDocument();

    const expandOutputMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} output1`,
    });
    fireEvent.click(expandOutputMappingButton);

    fireEvent.click(await screen.findByRole("button", { name: "output1" }));
    expect(
      await screen.findByRole("option", { name: "out1" })
    ).toBeInTheDocument();
  });

  test("deduplicates default mappings", async () => {
    const step = createFlowStep();
    stepsInStore = [
      {
        id: "child-step",
        name: "Child Step",
        type: "service",
        attributes: {},
        http: {
          invoke: { endpoint: "http://localhost:8080/child", timeout: 5000 },
        },
      },
    ];

    const { api } = jest.requireMock("@/app/api");
    const plan = {
      steps: {
        "child-step": {
          id: "child-step",
          name: "Child Step",
          type: "service",
          attributes: {
            input1: {
              role: AttributeRole.Required,
              type: AttributeType.String,
            },
            output1: {
              role: AttributeRole.Output,
              type: AttributeType.String,
            },
          },
        },
      },
      required: [],
      attributes: {},
      goals: ["child-step"],
    };
    api.getExecutionPlan.mockResolvedValue(plan);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    fireEvent.click(
      await screen.findByRole("button", { name: t("stepEditor.typeFlowLabel") })
    );
    await waitFor(() => {
      const flowButton = screen.getByTitle(t("stepEditor.typeFlowTitle"));
      fireEvent.click(flowButton);
    });

    await waitFor(() => {
      const goalChip = screen.getByText("child-step");
      fireEvent.click(goalChip);
    });

    const expandInputMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} input1`,
    });
    fireEvent.click(expandInputMappingButton);

    fireEvent.click(screen.getByRole("button", { name: "input1" }));
    expect(screen.getAllByRole("option", { name: "input1" })).toHaveLength(1);

    const expandOutputMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} output1`,
    });
    fireEvent.click(expandOutputMappingButton);

    fireEvent.click(screen.getByRole("button", { name: "output1" }));
    expect(screen.getAllByRole("option", { name: "output1" })).toHaveLength(1);
  });

  test("updates predicate", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const predicateInput = screen.getByDisplayValue(
        "return temperature > 100"
      );
      fireEvent.change(predicateInput, {
        target: { value: "(< temperature 50)" },
      });
      expect(
        screen.getByDisplayValue("(< temperature 50)")
      ).toBeInTheDocument();
    });
  });

  test("adds attribute via add button", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const addButton = screen.getByTitle(t("stepEditor.addAttribute"));
      fireEvent.click(addButton);
    });

    await waitFor(() => {
      const inputs = screen.getAllByPlaceholderText("name");
      expect(inputs.length).toBe(4);
    });
  });

  test("removes attribute via remove button", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const removeButtons = screen.getAllByTitle("Remove attribute");
      const initialCount = removeButtons.length;
      fireEvent.click(removeButtons[0]);

      waitFor(() => {
        const updatedButtons = screen.getAllByTitle("Remove attribute");
        expect(updatedButtons.length).toBe(initialCount - 1);
      });
    });
  });

  test("saves updated step successfully", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          name: "Test HTTP Step",
          type: "service",
          http: expect.objectContaining({
            invoke: expect.objectContaining({
              endpoint: "http://localhost:8080/test",
              timeout: expect.any(Number),
            }),
            health: "http://localhost:8080/health",
          }),
          predicate: expect.objectContaining({
            language: "lua",
            script: "return temperature > 100",
          }),
        })
      );
      expect(mockOnUpdate).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("saves attribute mapping edits", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    const expandMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} input1`,
    });
    fireEvent.click(expandMappingButton);

    const sourceInput = await screen.findByPlaceholderText("input1");
    fireEvent.change(sourceInput, {
      target: { value: "request_payload" },
    });

    const scriptInput = await screen.findByPlaceholderText(
      t("stepEditor.mappingScriptPlaceholderLua")
    );
    fireEvent.change(scriptInput, { target: { value: "$.payload.input" } });

    const saveButton = screen.getByText(t("stepEditor.save"));
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          attributes: expect.objectContaining({
            input1: expect.objectContaining({
              required: {
                mapping: {
                  name: "request_payload",
                  script: {
                    language: "lua",
                    script: "$.payload.input",
                  },
                },
              },
            }),
          }),
        })
      );
    });
  });

  test("uses mapping placeholders", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    const expandMappingButton = await screen.findByRole("button", {
      name: `${t("stepEditor.mappingLabel")} input1`,
    });
    fireEvent.click(expandMappingButton);

    expect(
      await screen.findByPlaceholderText(
        t("stepEditor.mappingScriptPlaceholderLua")
      )
    ).toBeInTheDocument();

    const mappingLanguageSelect = screen.getByLabelText(
      t("stepEditor.mappingLanguageLabel")
    );

    fireEvent.click(mappingLanguageSelect);
    fireEvent.click(screen.getByRole("option", { name: "JPath" }));
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText(
          t("stepEditor.mappingScriptPlaceholderJPath")
        )
      ).toBeInTheDocument();
    });

    fireEvent.click(mappingLanguageSelect);
    fireEvent.click(screen.getByRole("option", { name: "Lua" }));
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText(t("stepEditor.mappingScriptPlaceholderLua"))
      ).toBeInTheDocument();
    });
  });

  test("saves edits from JSON mode", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    const jsonModeButton = await screen.findByRole("button", {
      name: t("stepEditor.modeJson"),
    });
    fireEvent.click(jsonModeButton);

    const jsonInput = await screen.findByTestId("script-editor");

    fireEvent.change(jsonInput, {
      target: {
        value: JSON.stringify(
          {
            ...step,
            name: "Updated via JSON",
          },
          null,
          2
        ),
      },
    });

    fireEvent.click(screen.getByText(t("stepEditor.save")));

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          name: "Updated via JSON",
        })
      );
    });
  });

  test("shows error when endpoint is empty", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const endpointInput = screen.getByDisplayValue(
        "http://localhost:8080/test"
      );
      fireEvent.change(endpointInput, { target: { value: "" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.endpointRequired"))
      ).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("shows error when timeout is invalid", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const durationInputs = screen.getAllByTestId("duration-input");
      const httpTimeoutInput = durationInputs[durationInputs.length - 1];
      fireEvent.change(httpTimeoutInput, { target: { value: "0" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.timeoutPositive"))
      ).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("shows error when timeout is not a number", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const durationInputs = screen.getAllByTestId("duration-input");
      const httpTimeoutInput = durationInputs[durationInputs.length - 1];
      fireEvent.change(httpTimeoutInput, { target: { value: "" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.timeoutPositive"))
      ).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("handles API error on save", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockRejectedValue(new Error("Server error"));

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText("Server error")).toBeInTheDocument();
      expect(mockOnUpdate).not.toHaveBeenCalled();
      expect(mockOnClose).not.toHaveBeenCalled();
    });
  });

  test("handles generic error on save", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockRejectedValue(new Error("Network error"));

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeInTheDocument();
    });
  });

  test("closes modal on cancel", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const cancelButton = screen.getByText(t("stepEditor.cancel"));
      fireEvent.click(cancelButton);
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("closes modal on backdrop click", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const dialog = document.querySelector("dialog");
      fireEvent.click(dialog!);
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("does not close modal on content click", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const body = document.querySelector("dialog > div:nth-child(2)");
      fireEvent.click(body!);
      expect(mockOnClose).not.toHaveBeenCalled();
    });
  });

  test("closes modal when the dialog closes", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const dialog = document.querySelector("dialog");
      fireEvent(dialog!, new Event("close"));
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("disables buttons while saving", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText(t("stepEditor.saving"))).toBeInTheDocument();
      const cancelButton = screen.getByText(t("stepEditor.cancel"));
      expect(cancelButton).toBeDisabled();
    });
  });

  test("handles empty predicate", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const predicateInput = screen.getByDisplayValue(
        "return temperature > 100"
      );
      fireEvent.change(predicateInput, { target: { value: "" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          predicate: undefined,
        })
      );
    });
  });

  test("handles empty health check", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const healthInput = screen.getByDisplayValue(
        "http://localhost:8080/health"
      );
      fireEvent.change(healthInput, { target: { value: "" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          http: expect.objectContaining({
            invoke: expect.objectContaining({
              endpoint: "http://localhost:8080/test",
              timeout: expect.any(Number),
            }),
            health: undefined,
          }),
        })
      );
    });
  });

  test("updates optional arg deadline", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("input2")).toBeInTheDocument();
    });

    expect(
      screen.getAllByTestId("duration-input").length
    ).toBeGreaterThanOrEqual(1);
  });

  test("opens the dialog", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(document.querySelector("dialog")).toHaveAttribute("open");
    });
  });

  test("uses diagram container sizing", async () => {
    const step = createHttpStep();
    const div = document.createElement("div");
    Object.defineProperty(div, "getBoundingClientRect", {
      value: () => ({ width: 1000, height: 800 }),
    });
    const containerRef = {
      current: div,
    } as React.RefObject<HTMLDivElement>;

    render(
      <StepEditor
        step={step}
        onClose={mockOnClose}
        onUpdate={mockOnUpdate}
        diagramContainerRef={containerRef}
      />
    );

    await waitFor(() => {
      const dialog = document.querySelector("dialog") as HTMLElement;
      expect(dialog).toHaveStyle({ width: "800px", height: "691.2px" });
    });
  });

  test("updates script code", async () => {
    const step = createScriptStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const scriptEditors = screen.getAllByTestId("script-editor");
      const scriptCodeEditor = scriptEditors.find(
        (e) => (e as HTMLTextAreaElement).value === "return {result = 42}"
      ) as HTMLTextAreaElement;
      fireEvent.change(scriptCodeEditor, {
        target: { value: "return {result = 100}" },
      });
      expect(
        screen.getByDisplayValue("return {result = 100}")
      ).toBeInTheDocument();
    });
  });

  test("saves script step successfully", async () => {
    const step = createScriptStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-2",
        expect.objectContaining({
          type: "script",
          script: {
            language: "lua",
            script: "return {result = 42}",
          },
        })
      );
      expect(mockOnUpdate).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("shows error when script code is empty", async () => {
    const step = createScriptStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const scriptEditors = screen.getAllByTestId("script-editor");
      const scriptCodeEditor = scriptEditors.find(
        (e) => (e as HTMLTextAreaElement).value === "return {result = 42}"
      ) as HTMLTextAreaElement;
      fireEvent.change(scriptCodeEditor, { target: { value: "" } });

      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText(t("stepEditor.scriptRequired"))
      ).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("switches from HTTP to script type", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    fireEvent.click(
      await screen.findByRole("button", {
        name: t("stepEditor.typeServiceLabel"),
      })
    );
    await waitFor(() => {
      const scriptButton = screen.getByTitle(t("stepEditor.typeScriptTitle"));
      fireEvent.click(scriptButton);
    });

    await waitFor(() => {
      expect(screen.getByText(t("stepEditor.scriptLabel"))).toBeInTheDocument();
      expect(
        screen.queryByPlaceholderText("http://localhost:8080/process")
      ).not.toBeInTheDocument();
    });
  });

  test("switches from script to HTTP type", async () => {
    const step = createScriptStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    fireEvent.click(
      await screen.findByRole("button", {
        name: t("stepEditor.typeScriptLabel"),
      })
    );
    await waitFor(() => {
      const syncButton = screen.getByTitle(t("stepEditor.typeServiceTitle"));
      fireEvent.click(syncButton);
    });

    await waitFor(() => {
      expect(
        screen.getByPlaceholderText("http://localhost:8080/process")
      ).toBeInTheDocument();
      expect(
        screen.queryByText(t("stepEditor.scriptLabel"))
      ).not.toBeInTheDocument();
    });
  });

  test("clears script from HTTP steps", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          script: undefined,
        })
      );
    });
  });

  test("clears HTTP from script steps", async () => {
    const step = createScriptStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText(t("stepEditor.save"));
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-2",
        expect.objectContaining({
          http: undefined,
        })
      );
    });
  });
});
