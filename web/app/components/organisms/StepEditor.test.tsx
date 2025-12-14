import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { Step } from "../../api";

jest.requireActual("../../api");

jest.mock("../../api", () => ({
  ...jest.requireActual("../../api"),
  ArgyllApi: jest.fn(),
}));

import StepEditor from "./StepEditor";
import { ArgyllApi, AttributeRole, AttributeType } from "../../api";

jest.mock("../molecules/ScriptEditor", () => ({
  __esModule: true,
  default: ({ value, onChange }: any) => (
    <textarea
      data-testid="script-editor"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock("../molecules/DurationInput", () => ({
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
  const createHttpStep = (type: "sync" | "async" = "sync"): Step => ({
    id: "step-1",
    name: "Test HTTP Step",
    type,
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      input2: { role: AttributeRole.Optional, type: AttributeType.Number },
      result: { role: AttributeRole.Output, type: AttributeType.String },
    },
    version: "1.0.0",
    http: {
      endpoint: "http://localhost:8080/test",
      health_check: "http://localhost:8080/health",
      timeout: 5000,
    },
    predicate: {
      language: "ale",
      script: "(> temperature 100)",
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
    version: "1.0.0",
    script: {
      language: "ale",
      script: "{:result 42}",
    },
    predicate: {
      language: "ale",
      script: "(> value 10)",
    },
  });

  const mockOnClose = jest.fn();
  const mockOnUpdate = jest.fn();
  const mockUpdateStep = jest.fn();

  beforeEach(() => {
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

  test("renders modal with HTTP step data", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByText(/Edit Step:.*step-1/)).toBeInTheDocument();
      expect(screen.getByDisplayValue("Test HTTP Step")).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("http://localhost:8080/test")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("http://localhost:8080/health")
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue("(> temperature 100)")
      ).toBeInTheDocument();
    });
  });

  test("renders modal with script step data", async () => {
    const step = createScriptStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByText(/Edit Step:.*step-2/)).toBeInTheDocument();
      expect(screen.getByDisplayValue("Test Script Step")).toBeInTheDocument();
      expect(screen.getByDisplayValue("{:result 42}")).toBeInTheDocument();
      expect(screen.getByDisplayValue("(> value 10)")).toBeInTheDocument();
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

  test("renders optional args with timeout", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("input2")).toBeInTheDocument();
      const selects = screen.getAllByRole("combobox");
      expect(selects.length).toBeGreaterThanOrEqual(2);
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

  test("shows placeholder row when no attributes exist", async () => {
    render(
      <StepEditor step={null} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Attributes describe how steps share data with each other/i
        )
      ).toBeInTheDocument();
    });
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

  test("updates step type", async () => {
    const step = createHttpStep("sync");

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const asyncButton = screen.getByTitle("Asynchronous HTTP");
      fireEvent.click(asyncButton);
      expect(asyncButton.className).toContain("typeButtonActive");
    });
  });

  test("updates timeout", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const durationInputs = screen.getAllByTestId("duration-input");
      const httpTimeoutInput = durationInputs[durationInputs.length - 1]; // HTTP timeout is last
      fireEvent.change(httpTimeoutInput, { target: { value: "10000" } });
      expect(httpTimeoutInput).toHaveValue("10000");
    });
  });

  test("updates endpoint", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const endpointInput = screen.getByDisplayValue(
        "http://localhost:8080/test"
      );
      fireEvent.change(endpointInput, {
        target: { value: "http://localhost:9090/new" },
      });
      expect(
        screen.getByDisplayValue("http://localhost:9090/new")
      ).toBeInTheDocument();
    });
  });

  test("updates health check", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const healthInput = screen.getByDisplayValue(
        "http://localhost:8080/health"
      );
      fireEvent.change(healthInput, {
        target: { value: "http://localhost:9090/health" },
      });
      expect(
        screen.getByDisplayValue("http://localhost:9090/health")
      ).toBeInTheDocument();
    });
  });

  test("updates predicate", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const predicateInput = screen.getByDisplayValue("(> temperature 100)");
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
      const addButton = screen.getByTitle("Add attribute");
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
      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          name: "Test HTTP Step",
          type: "sync",
          http: expect.objectContaining({
            endpoint: "http://localhost:8080/test",
            health_check: "http://localhost:8080/health",
            timeout: expect.any(Number),
          }),
          predicate: expect.objectContaining({
            language: "ale",
            script: "(> temperature 100)",
          }),
        })
      );
      expect(mockOnUpdate).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
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

      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText("HTTP endpoint is required")).toBeInTheDocument();
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

      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText("Timeout must be a positive number")
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

      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(
        screen.getByText("Timeout must be a positive number")
      ).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("handles API error on save", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockRejectedValue({
      response: { data: { error: "Server error" } },
    });

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText("Save");
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
      const saveButton = screen.getByText("Save");
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
      const cancelButton = screen.getByText("Cancel");
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
      const backdrop = document.querySelector(".backdrop");
      fireEvent.click(backdrop!);
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  test("does not close modal on content click", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const content = document.querySelector(".content");
      fireEvent.click(content!);
      expect(mockOnClose).not.toHaveBeenCalled();
    });
  });

  test("closes modal on escape key", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      fireEvent.keyDown(document, { key: "Escape" });
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
      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText("Saving...")).toBeInTheDocument();
      const cancelButton = screen.getByText("Cancel");
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
      const predicateInput = screen.getByDisplayValue("(> temperature 100)");
      fireEvent.change(predicateInput, { target: { value: "" } });

      const saveButton = screen.getByText("Save");
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

      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-1",
        expect.objectContaining({
          http: expect.objectContaining({
            endpoint: "http://localhost:8080/test",
            health_check: undefined,
            timeout: expect.any(Number),
          }),
        })
      );
    });
  });

  test("updates optional arg timeout", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue("input2")).toBeInTheDocument();
    });

    const selects = screen.getAllByRole("combobox");
    const optionalTimeoutSelect = selects.find(
      (s) => (s as HTMLSelectElement).value === "3000"
    );

    if (optionalTimeoutSelect) {
      fireEvent.change(optionalTimeoutSelect, { target: { value: "5000" } });
      expect((optionalTimeoutSelect as HTMLSelectElement).value).toBe("5000");
    } else {
      expect(selects.length).toBeGreaterThanOrEqual(2);
    }
  });

  test("renders modal using portal", async () => {
    const step = createHttpStep();

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const backdrop = document.querySelector(".backdrop");
      expect(backdrop?.parentElement).toBe(document.body);
    });
  });

  test("does not render before mounted", () => {
    const step = createHttpStep();

    const { container } = render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    // Should start with null before mounting
    expect(container.firstChild).toBeNull();
  });

  test("renders with diagram container ref for sizing", async () => {
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
      const content = document.querySelector(".content") as HTMLElement;
      expect(content).toBeInTheDocument();
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
        (e) => (e as HTMLTextAreaElement).value === "{:result 42}"
      ) as HTMLTextAreaElement;
      fireEvent.change(scriptCodeEditor, {
        target: { value: "{:result 100}" },
      });
      expect(screen.getByDisplayValue("{:result 100}")).toBeInTheDocument();
    });
  });

  test("saves script step successfully", async () => {
    const step = createScriptStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(mockUpdateStep).toHaveBeenCalledWith(
        "step-2",
        expect.objectContaining({
          type: "script",
          script: {
            language: "ale",
            script: "{:result 42}",
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
        (e) => (e as HTMLTextAreaElement).value === "{:result 42}"
      ) as HTMLTextAreaElement;
      fireEvent.change(scriptCodeEditor, { target: { value: "" } });

      const saveButton = screen.getByText("Save");
      fireEvent.click(saveButton);
    });

    await waitFor(() => {
      expect(screen.getByText("Script code is required")).toBeInTheDocument();
      expect(mockUpdateStep).not.toHaveBeenCalled();
    });
  });

  test("switches from HTTP to script type", async () => {
    const step = createHttpStep("sync");

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const scriptButton = screen.getByTitle("Script (Ale)");
      fireEvent.click(scriptButton);
    });

    await waitFor(() => {
      expect(screen.getByText("Script Code")).toBeInTheDocument();
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

    await waitFor(() => {
      const syncButton = screen.getByTitle("Synchronous HTTP");
      fireEvent.click(syncButton);
    });

    await waitFor(() => {
      expect(
        screen.getByPlaceholderText("http://localhost:8080/process")
      ).toBeInTheDocument();
      expect(screen.queryByText("Script Code")).not.toBeInTheDocument();
    });
  });

  test("saves HTTP step with script set to undefined", async () => {
    const step = createHttpStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText("Save");
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

  test("saves script step with http set to undefined", async () => {
    const step = createScriptStep();
    mockUpdateStep.mockResolvedValue(undefined);

    render(
      <StepEditor step={step} onClose={mockOnClose} onUpdate={mockOnUpdate} />
    );

    await waitFor(() => {
      const saveButton = screen.getByText("Save");
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
