import { render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
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
    type: "service" | "script" | "flow",
    config?: any
  ): Step => ({
    id: "step-1",
    name: "Test Step",
    type,
    attributes: {},

    ...(type === "script"
      ? {
          script: config || {
            language: "lua",
            source: "return {result = 1 + 2}",
          },
        }
      : type === "flow"
        ? {
            flow: {
              goals: config?.goals || { steps: ["goal-a", "goal-b"] },
            },
          }
        : {
            http: {
              invoke: {
                endpoint: "http://localhost:8080/test",
                timeout: 5000,
                ...config,
              },
            },
          }),
  });

  beforeEach(() => {
    mockUseStepProgress.mockReturnValue({
      status: "pending",
    });
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.clearAllMocks();
  });

  test("renders HTTP endpoint for sync step", () => {
    const step = createStep("service", {
      endpoint: "http://localhost:8080/process",
    });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("POST http://localhost:8080/process");
  });

  test("renders HTTP endpoint for async invoke steps", () => {
    const step = createStep("service", {
      endpoint: "http://localhost:8080/async",
      mode: "async",
    });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("POST http://localhost:8080/async");
  });

  test("renders configured HTTP method for http steps", () => {
    const step = createStep("service", {
      endpoint: "http://localhost:8080/items/{item_id}",
      method: "GET",
    });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe(
      "GET http://localhost:8080/items/{item_id}"
    );
  });

  test("renders script preview for script step", () => {
    const step = createStep("script", {
      language: "lua",
      source: 'return {greeting = "Hello" .. name}',
    });

    const { container } = render(<Footer step={step} />);

    const preview = container.querySelector(".step-endpoint");
    expect(preview).toBeInTheDocument();
    expect(preview?.textContent).toBe('{greeting = "Hello" .. name}');
  });

  test("renders flow goals for flow step", () => {
    const step = createStep("flow", {
      goals: { steps: ["goal-a", "goal-b"], else: { steps: ["goal-c"] } },
    });

    const { container } = render(<Footer step={step} />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("goal-a, goal-b ELSE goal-c");
    expect(screen.getByText(t("stepFooter.flowGoals"))).toBeInTheDocument();
    const tooltipSections = screen.getAllByTestId("tooltip-section");
    const hasGoalsSection = tooltipSections.some((section) =>
      section.textContent?.includes("goal-a, goal-b ELSE goal-c")
    );
    expect(hasGoalsSection).toBe(true);
  });

  test("replaces newlines in script preview", () => {
    const step = createStep("script", {
      language: "lua",
      source: "return {\n  result = 1 + 2\n}",
    });

    const { container } = render(<Footer step={step} />);

    const preview = container.querySelector(".step-endpoint");
    expect(preview?.textContent).toBe("{ result = 1 + 2 }");
  });

  test("shows progress icon when flow is active", () => {
    const step = createStep("service");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-1",
    });

    const { container } = render(<Footer step={step} flowId="wf-1" />);

    expect(container.querySelector(".progress-icon")).toBeInTheDocument();
  });

  test("renders execution status in tooltip", () => {
    const step = createStep("service");
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

    expect(screen.getByText(t("liveStep.executionStatus"))).toBeInTheDocument();
    expect(screen.getByText("COMPLETED")).toBeInTheDocument();
  });

  test("shows active work timeout progress in tooltip", () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date("2024-01-01T00:00:02Z"));

    const step = createStep("service", { timeout: 5000 });
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "active",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      work_items: {
        "token-1": {
          token: "token-1",
          status: "active",
          started_at: "2024-01-01T00:00:00Z",
          inputs: {},
          retry_count: 0,
        },
      },
    };

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(
      screen.getByText(t("liveStep.activeTimeoutTitle"))
    ).toBeInTheDocument();
    expect(screen.getByText("3s")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("value", "40");
  });

  test("shows pending retry countdown in tooltip", () => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date("2024-01-01T00:00:02Z"));

    const step = createStep("service");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "active",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      work_items: {
        "token-1": {
          token: "token-1",
          status: "pending",
          started_at: "2024-01-01T00:00:00Z",
          completed_at: "2024-01-01T00:00:00Z",
          inputs: {},
          retry_count: 1,
          next_retry_at: "2024-01-01T00:00:05Z",
        },
      },
    };

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(
      screen.getByText(t("liveStep.retryCountdownTitle"))
    ).toBeInTheDocument();
    expect(screen.getByText("3s")).toBeInTheDocument();
    expect(screen.getByRole("progressbar")).toHaveAttribute("value", "40");
  });

  test("shows error message for failed execution", () => {
    const step = createStep("service");
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

    expect(screen.getByText(t("liveStep.errorTitle"))).toBeInTheDocument();
    expect(screen.getByText("Connection timeout")).toBeInTheDocument();
  });

  test("shows generic skip reason when skipped execution has no reason", () => {
    const step = createStep("service");
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

    expect(screen.getByText(t("liveStep.reasonTitle"))).toBeInTheDocument();
    expect(screen.getByText(t("liveStep.skipUnknown"))).toBeInTheDocument();
  });

  test("shows required match reason for required match skip", () => {
    const step = createStep("service");
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "skipped",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      error_message: "required match did not match",
    };

    mockUseStepProgress.mockReturnValue({
      status: "skipped",
      flowId: "wf-1",
    });

    render(<Footer step={step} flowId="wf-1" execution={execution} />);

    expect(
      screen.getByText(t("liveStep.skipRequiredMatch"))
    ).toBeInTheDocument();
  });

  test("shows duration for completed execution", () => {
    const step = createStep("service");
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

    expect(screen.getByText(t("liveStep.durationTitle"))).toBeInTheDocument();
    expect(
      screen.getByText(t("common.durationMs", { duration: 2500 }))
    ).toBeInTheDocument();
  });

  test("shows reported Agent usage in a table after duration", () => {
    const step: Step = {
      id: "step-1",
      name: "Agent Step",
      type: "agent",
      attributes: {},
    };
    const work = { status: "succeeded" as const, inputs: {}, retry_count: 0 };
    const execution: ExecutionResult = {
      step_id: step.id,
      flow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
      duration_ms: 2500,
      work_items: {
        a: {
          ...work,
          token: "a",
          usage: {
            cost: 0.00001,
            cost_calculated: 0.00001,
            input_tokens: 500,
            output_tokens: 20,
            cache_input_tokens: 150,
            cache_output_tokens: 50,
            provider_metric: 2,
          },
        },
        b: {
          ...work,
          token: "b",
          usage: {
            input_tokens: 100,
            cost_unpriced_tokens: 100,
            provider_metric: 3,
          },
        },
        c: {
          ...work,
          token: "c",
          usage: { cost: 0.00002, cost_reported: 0.00002 },
        },
      },
    };

    const { rerender } = render(
      <Footer step={step} flowId="wf-1" execution={execution} />
    );

    expect(
      screen.getByRole("table", { name: t("liveStep.usageTitle") })
    ).toBeInTheDocument();
    const cost = screen.getByRole("rowheader", {
      name: t("liveStep.usage.cost"),
    });
    expect(cost.nextElementSibling).toHaveTextContent("0.00003");
    expect(
      screen.getByRole("rowheader", { name: "Calculated Cost" })
        .nextElementSibling
    ).toHaveTextContent("0.00001");
    expect(
      screen.getByRole("rowheader", { name: "Provider-Reported Cost" })
        .nextElementSibling
    ).toHaveTextContent("0.00002");
    expect(
      screen.getByRole("rowheader", { name: "Unpriced Tokens" })
        .nextElementSibling
    ).toHaveTextContent("100");
    expect(
      screen.getByText(t("liveStep.usageTitle")).parentElement
        ?.previousElementSibling
    ).toHaveTextContent(t("liveStep.durationTitle"));
    expect(
      screen.getByText(t("liveStep.usage.input_tokens")).nextElementSibling
    ).toHaveTextContent("600");
    expect(
      screen.getByText(t("liveStep.usage.output_tokens")).nextElementSibling
    ).toHaveTextContent("20");
    expect(
      screen.getByText(t("liveStep.usage.cache_input_tokens"))
        .nextElementSibling
    ).toHaveTextContent("150");
    expect(
      screen.getByText(t("liveStep.usage.cache_output_tokens"))
        .nextElementSibling
    ).toHaveTextContent("50");
    expect(
      screen.getByRole("rowheader", { name: "provider_metric" })
        .nextElementSibling
    ).toHaveTextContent("5");

    rerender(
      <Footer
        step={step}
        flowId="wf-1"
        execution={{
          ...execution,
          work_items: { a: { ...work, token: "a", usage: { cost: 0 } } },
        }}
      />
    );
    expect(cost.nextElementSibling).toHaveTextContent(/^0$/);

    rerender(
      <Footer
        step={step}
        flowId="wf-1"
        execution={{ ...execution, work_items: {} }}
      />
    );
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  test("handles step with no http or script", () => {
    const step: Step = {
      id: "step-1",
      name: "Test",
      type: "service",
      attributes: {},
    };

    const { container } = render(<Footer step={step} />);

    expect(container.querySelector(".step-endpoint")).not.toBeInTheDocument();
  });

  test("does not show progress icon when flow IDs don't match", () => {
    const step = createStep("service");

    mockUseStepProgress.mockReturnValue({
      status: "active",
      flowId: "wf-2",
    });

    const { container } = render(<Footer step={step} flowId="wf-1" />);

    expect(container.querySelector(".progress-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("tooltip")).toBeInTheDocument();
  });
});
