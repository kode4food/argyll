import React from "react";
import { render, screen } from "@testing-library/react";
import StepFooter from "./StepFooter";
import type { Step, ExecutionResult } from "../../api";
import { useStepProgress } from "../../hooks/useStepProgress";

jest.mock("../../hooks/useStepProgress");
jest.mock("../atoms/Tooltip", () => ({
  __esModule: true,
  default: ({ trigger, children }: any) => (
    <div data-testid="tooltip">
      {trigger}
      <div data-testid="tooltip-content">{children}</div>
    </div>
  ),
}));
jest.mock("../atoms/TooltipSection", () => ({
  __esModule: true,
  default: ({ children, title }: any) => (
    <div data-testid="tooltip-section">
      <div>{title}</div>
      <div>{children}</div>
    </div>
  ),
}));
jest.mock("../atoms/HealthDot", () => ({
  __esModule: true,
  default: ({ className }: any) => (
    <div data-testid="health-dot" className={className} />
  ),
}));

const mockUseStepProgress = useStepProgress as jest.MockedFunction<
  typeof useStepProgress
>;

describe("StepFooter", () => {
  const createStep = (
    type: "sync" | "async" | "script",
    config?: any
  ): Step => ({
    id: "step-1",
    name: "Test Step",
    type,
    attributes: {},

    version: "1.0.0",
    ...(type === "script"
      ? {
          script: config || {
            language: "ale",
            script: "{:result (+ 1 2)}",
          },
        }
      : {
          http: {
            endpoint: "http://localhost:8080/test",
            timeout: 5000,
            ...config,
          },
        }),
  });

  beforeEach(() => {
    mockUseStepProgress.mockReturnValue({
      status: "pending",
    });
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  test("renders HTTP endpoint for sync step", () => {
    const step = createStep("sync", {
      endpoint: "http://localhost:8080/process",
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" />
    );

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("http://localhost:8080/process");
  });

  test("renders HTTP endpoint for async step with webhook icon", () => {
    const step = createStep("async", {
      endpoint: "http://localhost:8080/async",
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" />
    );

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("http://localhost:8080/async");
  });

  test("renders script preview for script step", () => {
    const step = createStep("script", {
      language: "ale",
      script: '{:greeting (str "Hello" name)}',
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" />
    );

    const preview = container.querySelector(".step-endpoint");
    expect(preview).toBeInTheDocument();
    expect(preview?.textContent).toBe('{:greeting (str "Hello" name)}');
  });

  test("replaces newlines in script preview", () => {
    const step = createStep("script", {
      language: "ale",
      script: "{\n  :result\n  (+ 1 2)\n}",
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" />
    );

    const preview = container.querySelector(".step-endpoint");
    expect(preview?.textContent).toBe("{   :result   (+ 1 2) }");
  });

  test("shows health dot when no workflow", () => {
    const step = createStep("sync");

    render(<StepFooter step={step} healthStatus="healthy" />);

    expect(screen.getByTestId("health-dot")).toBeInTheDocument();
  });

  test("shows progress icon when workflow is active", () => {
    const step = createStep("sync");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      workflowId: "wf-1",
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" workflowId="wf-1" />
    );

    expect(container.querySelector(".progress-icon")).toBeInTheDocument();
  });

  test("renders execution status in tooltip", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      duration_ms: 1500,
    };

    mockUseStepProgress.mockReturnValue({
      status: "completed",
      workflowId: "wf-1",
    });

    render(
      <StepFooter
        step={step}
        healthStatus="healthy"
        workflowId="wf-1"
        execution={execution}
      />
    );

    expect(screen.getByText("Execution Status")).toBeInTheDocument();
    expect(screen.getByText("COMPLETED")).toBeInTheDocument();
  });

  test("shows error message for failed execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "failed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      error_message: "Connection timeout",
    };

    mockUseStepProgress.mockReturnValue({
      status: "failed",
      workflowId: "wf-1",
    });

    render(
      <StepFooter
        step={step}
        healthStatus="healthy"
        workflowId="wf-1"
        execution={execution}
      />
    );

    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getByText("Connection timeout")).toBeInTheDocument();
  });

  test("shows skip reason for skipped execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "skipped",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    mockUseStepProgress.mockReturnValue({
      status: "skipped",
      workflowId: "wf-1",
    });

    render(
      <StepFooter
        step={step}
        healthStatus="healthy"
        workflowId="wf-1"
        execution={execution}
      />
    );

    expect(screen.getByText("Reason")).toBeInTheDocument();
    expect(
      screen.getByText(/Step skipped because required inputs are unavailable/)
    ).toBeInTheDocument();
  });

  test("shows duration for completed execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      workflow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      duration_ms: 2500,
    };

    mockUseStepProgress.mockReturnValue({
      status: "completed",
      workflowId: "wf-1",
    });

    render(
      <StepFooter
        step={step}
        healthStatus="healthy"
        workflowId="wf-1"
        execution={execution}
      />
    );

    expect(screen.getByText("Duration")).toBeInTheDocument();
    expect(screen.getByText("2500ms")).toBeInTheDocument();
  });

  test("shows health check URL in tooltip for HTTP steps", () => {
    const step = createStep("sync", {
      endpoint: "http://localhost:8080/process",
      health_check: "http://localhost:8080/health",
    });

    render(<StepFooter step={step} healthStatus="healthy" />);

    expect(screen.getByText("Health Check URL")).toBeInTheDocument();
    expect(
      screen.getByText("http://localhost:8080/health")
    ).toBeInTheDocument();
  });

  test("shows script preview in tooltip for script steps", () => {
    const multilineScript = `{:result
  (+ 1 2)
  (+ 3 4)
  (+ 5 6)
  (+ 7 8)}`;

    const step = createStep("script", {
      language: "ale",
      script: multilineScript,
    });

    render(<StepFooter step={step} healthStatus="healthy" />);

    expect(screen.getByText(/Script Preview/)).toBeInTheDocument();
  });

  test("shows health status in tooltip when no workflow", () => {
    const step = createStep("sync");

    render(
      <StepFooter
        step={step}
        healthStatus="unhealthy"
        healthError="Service unavailable"
      />
    );

    expect(screen.getByText("Health Status")).toBeInTheDocument();
  });

  test("handles step with no http or script", () => {
    const step: Step = {
      id: "step-1",
      name: "Test",
      type: "sync",
      attributes: {},

      version: "1.0.0",
    };

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" />
    );

    expect(container.querySelector(".step-endpoint")).not.toBeInTheDocument();
  });

  test("does not show progress icon when workflow IDs don't match", () => {
    const step = createStep("sync");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      workflowId: "wf-2",
    });

    const { container } = render(
      <StepFooter step={step} healthStatus="healthy" workflowId="wf-1" />
    );

    expect(container.querySelector(".progress-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("health-dot")).toBeInTheDocument();
  });
});
