import { render, screen } from "@testing-library/react";
import Footer from "./Footer";
import type { Step, ExecutionResult } from "@/app/api";
import { useStepProgress } from "@/app/hooks/useStepProgress";

jest.mock("@/app/hooks/useStepProgress");
jest.mock("@/app/components/atoms/Tooltip", () => ({
  __esModule: true,
  default: ({ trigger, children }: any) => (
    <div data-testid="tooltip">
      {trigger}
      <div data-testid="tooltip-content">{children}</div>
    </div>
  ),
}));
jest.mock("@/app/components/atoms/TooltipSection", () => ({
  __esModule: true,
  default: ({ children, title }: any) => (
    <div data-testid="tooltip-section">
      <div>{title}</div>
      <div>{children}</div>
    </div>
  ),
}));
jest.mock("@/app/components/atoms/HealthDot", () => ({
  __esModule: true,
  default: ({ className }: any) => (
    <div data-testid="health-dot" className={className} />
  ),
}));

const mockUseStepProgress = useStepProgress as jest.MockedFunction<
  typeof useStepProgress
>;

describe("Footer", () => {
  const createStep = (
    type: "sync" | "async" | "script" | "flow",
    config?: any
  ): Step => ({
    id: "step-1",
    name: "Test Step",
    type,
    attributes: {},

    ...(type === "script"
      ? {
          script: config || {
            language: "ale",
            script: "{:result (+ 1 2)}",
          },
        }
      : type === "flow"
        ? {
            flow: {
              goals: config?.goals || ["goal-a", "goal-b"],
              input_map: {},
              output_map: {},
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

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("http://localhost:8080/process");
  });

  test("renders HTTP endpoint for async step with webhook icon", () => {
    const step = createStep("async", {
      endpoint: "http://localhost:8080/async",
    });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("http://localhost:8080/async");
  });

  test("renders script preview for script step", () => {
    const step = createStep("script", {
      language: "ale",
      script: '{:greeting (str "Hello" name)}',
    });

    const { container } = render(<Footer step={step} />);

    const preview = container.querySelector(".step-endpoint");
    expect(preview).toBeInTheDocument();
    expect(preview?.textContent).toBe('{:greeting (str "Hello" name)}');
  });

  test("renders flow goals for flow step", () => {
    const step = createStep("flow", { goals: ["goal-a", "goal-b"] });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("goal-a, goal-b");
    expect(screen.getByText("Goal Steps")).toBeInTheDocument();
  });

  test("replaces newlines in script preview", () => {
    const step = createStep("script", {
      language: "ale",
      script: "{\n  :result\n  (+ 1 2)\n}",
    });

    const { container } = render(<Footer step={step} />);

    const preview = container.querySelector(".step-endpoint");
    expect(preview?.textContent).toBe("{   :result   (+ 1 2) }");
  });

  test("shows progress icon when flow is active", () => {
    const step = createStep("sync");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-1",
    });

    const { container } = render(<Footer step={step} flowId="wf-1" />);

    expect(container.querySelector(".progress-icon")).toBeInTheDocument();
  });

  test("renders execution status in tooltip", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      duration_ms: 1500,
    };

    mockUseStepProgress.mockReturnValue({
      status: "completed",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(screen.getByText("Execution Status")).toBeInTheDocument();
    expect(screen.getByText("COMPLETED")).toBeInTheDocument();
  });

  test("shows error message for failed execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "failed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      error_message: "Connection timeout",
    };

    mockUseStepProgress.mockReturnValue({
      status: "failed",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getByText("Connection timeout")).toBeInTheDocument();
  });

  test("shows skip reason for skipped execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "skipped",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    mockUseStepProgress.mockReturnValue({
      status: "skipped",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(screen.getByText("Reason")).toBeInTheDocument();
    expect(
      screen.getByText(/Step skipped because required inputs are unavailable/)
    ).toBeInTheDocument();
  });

  test("shows duration for completed execution", () => {
    const step = createStep("sync");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      duration_ms: 2500,
    };

    mockUseStepProgress.mockReturnValue({
      status: "completed",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(screen.getByText("Duration")).toBeInTheDocument();
    expect(screen.getByText("2500ms")).toBeInTheDocument();
  });

  test("handles step with no http or script", () => {
    const step: Step = {
      id: "step-1",
      name: "Test",
      type: "sync",
      attributes: {},
    };

    const { container } = render(<Footer step={step} />);

    expect(container.querySelector(".step-endpoint")).not.toBeInTheDocument();
  });

  test("does not show progress icon when flow IDs don't match", () => {
    const step = createStep("sync");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-2",
    });

    const { container } = render(<Footer step={step} flowId="wf-1" />);

    expect(container.querySelector(".progress-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("tooltip")).toBeInTheDocument();
  });
});
