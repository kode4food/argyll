import React from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import StepWidget from "./StepWidget";
import type { Step, ExecutionResult } from "../../api";
import { useStepHealth } from "../../hooks/useStepHealth";

jest.mock("../../hooks/useStepHealth");
jest.mock("../molecules/StepHeader", () => ({
  __esModule: true,
  default: ({ step }: any) => <div data-testid="step-header">{step.name}</div>,
}));
jest.mock("../molecules/StepAttributesSection", () => ({
  __esModule: true,
  default: ({ step }: any) => <div data-testid="step-args">{step.id}</div>,
}));
jest.mock("../molecules/StepPredicate", () => ({
  __esModule: true,
  default: () => null,
}));
jest.mock("../molecules/StepFooter", () => ({
  __esModule: true,
  default: ({ step }: any) => <div data-testid="step-footer">{step.id}</div>,
}));
jest.mock("../../contexts/StepEditorContext", () => {
  const openEditor = jest.fn();
  const closeEditor = jest.fn();
  return {
    __esModule: true,
    StepEditorProvider: ({ children }: { children: React.ReactNode }) =>
      children,
    useStepEditorContext: () => ({
      openEditor,
      closeEditor,
      isOpen: false,
      activeStep: null,
    }),
    __openEditor: openEditor,
    __closeEditor: closeEditor,
  };
});

const mockUseStepHealth = useStepHealth as jest.MockedFunction<
  typeof useStepHealth
>;

describe("StepWidget", () => {
  const createStep = (
    type: "sync" | "async" | "script",
    id: string = "step-1"
  ): Step => ({
    id,
    name: `Test Step ${id}`,
    type,
    attributes: {},

    ...(type === "script"
      ? {
          script: {
            language: "ale",
            script: "{:result 42}",
          },
        }
      : {
          http: {
            endpoint: "http://localhost:8080/test",
            timeout: 5000,
          },
        }),
  });

  beforeEach(() => {
    mockUseStepHealth.mockReturnValue({
      status: "healthy",
      error: undefined,
    });
    const {
      __openEditor,
      __closeEditor,
    } = require("../../contexts/StepEditorContext");
    __openEditor.mockClear();
    __closeEditor.mockClear();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("renders step components", () => {
    const step = createStep("sync");

    render(<StepWidget step={step} />);

    expect(screen.getByTestId("step-header")).toBeInTheDocument();
    expect(screen.getByTestId("step-args")).toBeInTheDocument();
    expect(screen.getByTestId("step-footer")).toBeInTheDocument();
  });

  test("applies selected className when selected", () => {
    const step = createStep("sync");

    const { container } = render(<StepWidget step={step} selected={true} />);

    const widget = container.querySelector(".step-widget");
    expect(widget?.className).toContain("selected");
  });

  test("applies clickable className when onClick provided", () => {
    const step = createStep("sync");
    const onClick = jest.fn();

    const { container } = render(<StepWidget step={step} onClick={onClick} />);

    const widget = container.querySelector(".step-widget");
    expect(widget?.className).toContain("clickable");
  });

  test("calls onClick when clicked", () => {
    const step = createStep("sync");
    const onClick = jest.fn();

    const { container } = render(<StepWidget step={step} onClick={onClick} />);

    const widget = container.querySelector(".step-widget");
    fireEvent.click(widget!);

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test("applies grayed-out className in preview mode when not in plan", () => {
    const step = createStep("sync");

    const { container } = render(
      <StepWidget step={step} isPreviewMode={true} isInPreviewPlan={false} />
    );

    const widget = container.querySelector(".step-widget");
    expect(widget?.className).toContain("grayed-out");
  });

  test("does not apply grayed-out className when in preview plan", () => {
    const step = createStep("sync");

    const { container } = render(
      <StepWidget step={step} isPreviewMode={true} isInPreviewPlan={true} />
    );

    const widget = container.querySelector(".step-widget");
    expect(widget?.className).not.toContain("grayed-out");
  });

  test("opens editor on double-click for script steps", async () => {
    const step = createStep("script");

    const { container } = render(<StepWidget step={step} />);

    const widget = container.querySelector(".step-widget");
    fireEvent.doubleClick(widget!);

    const { __openEditor } = require("../../contexts/StepEditorContext");
    expect(__openEditor).toHaveBeenCalled();
  });

  test("opens editor on double-click for HTTP steps", async () => {
    const step = createStep("sync");

    const { container } = render(<StepWidget step={step} />);

    const widget = container.querySelector(".step-widget");
    fireEvent.doubleClick(widget!);

    const { __openEditor } = require("../../contexts/StepEditorContext");
    expect(__openEditor).toHaveBeenCalled();
  });

  test("does not open editor when disableEdit is true", () => {
    const step = createStep("script");

    const { container } = render(<StepWidget step={step} disableEdit={true} />);

    const widget = container.querySelector(".step-widget");
    fireEvent.doubleClick(widget!);

    const { __openEditor } = require("../../contexts/StepEditorContext");
    expect(__openEditor).not.toHaveBeenCalled();
  });

  test("applies custom className", () => {
    const step = createStep("sync");

    const { container } = render(
      <StepWidget step={step} className="custom-class" />
    );

    const widget = container.querySelector(".step-widget");
    expect(widget?.className).toContain("custom-class");
  });

  test("applies custom style", () => {
    const step = createStep("sync");
    const style = { backgroundColor: "red" };

    const { container } = render(<StepWidget step={step} style={style} />);

    const widget = container.querySelector(".step-widget") as HTMLElement;
    expect(widget?.style.backgroundColor).toBe("red");
  });

  test("applies mode className", () => {
    const step = createStep("sync");

    const { container: listContainer } = render(
      <StepWidget step={step} mode="list" />
    );
    const { container: diagramContainer } = render(
      <StepWidget step={step} mode="diagram" />
    );

    const listWidget = listContainer.querySelector(".step-widget");
    const diagramWidget = diagramContainer.querySelector(".step-widget");

    expect(listWidget?.className).toContain("list");
    expect(diagramWidget?.className).toContain("diagram");
  });

  test("passes execution to child components", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    render(<StepWidget step={step} execution={execution} />);

    expect(screen.getByTestId("step-footer")).toBeInTheDocument();
  });

  test("passes satisfiedArgs to StepAttributesSection", () => {
    const step = createStep("sync");
    const satisfiedArgs = new Set(["arg1", "arg2"]);

    render(<StepWidget step={step} satisfiedArgs={satisfiedArgs} />);

    expect(screen.getByTestId("step-args")).toBeInTheDocument();
  });

  test("shows edit title for script steps when not disabled", () => {
    const step = createStep("script");

    const { container } = render(<StepWidget step={step} />);

    const widget = container.querySelector(".step-widget") as HTMLElement;
    expect(widget?.title).toBe("Double-click to edit step");
  });

  test("shows edit title for HTTP steps when not disabled", () => {
    const step = createStep("sync");

    const { container } = render(<StepWidget step={step} />);

    const widget = container.querySelector(".step-widget") as HTMLElement;
    expect(widget?.title).toBe("Double-click to edit step");
  });

  test("does not show edit title when edit is disabled", () => {
    const step = createStep("script");

    const { container } = render(<StepWidget step={step} disableEdit={true} />);

    const widget = container.querySelector(".step-widget") as HTMLElement;
    expect(widget?.title).toBe("");
  });

  test("listens for openStepEditor custom event", async () => {
    const step = createStep("script", "step-123");

    render(<StepWidget step={step} />);

    const event = new CustomEvent("openStepEditor", {
      detail: { stepId: "step-123" },
    });

    const { __openEditor } = require("../../contexts/StepEditorContext");
    await waitFor(() => {
      document.dispatchEvent(event);
      expect(__openEditor).toHaveBeenCalled();
    });
  });

  test("ignores openStepEditor event for different step", () => {
    const step = createStep("script", "step-123");

    render(<StepWidget step={step} />);

    const event = new CustomEvent("openStepEditor", {
      detail: { stepId: "step-456" },
    });
    document.dispatchEvent(event);

    const { __openEditor } = require("../../contexts/StepEditorContext");
    expect(__openEditor).not.toHaveBeenCalled();
  });

  test("ignores openStepEditor event when disabled", () => {
    const step = createStep("script", "step-123");

    render(<StepWidget step={step} disableEdit={true} />);

    const event = new CustomEvent("openStepEditor", {
      detail: { stepId: "step-123" },
    });
    document.dispatchEvent(event);

    const { __openEditor } = require("../../contexts/StepEditorContext");
    expect(__openEditor).not.toHaveBeenCalled();
  });
});
