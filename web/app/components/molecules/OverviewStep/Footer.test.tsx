import { render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import Footer from "./Footer";
import type { Step } from "@/app/api";

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

  test("renders HTTP endpoint for sync step", () => {
    const step = createStep("sync", {
      endpoint: "http://localhost:8080/process",
    });

    const { container } = render(<Footer step={step} healthStatus="healthy" />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("http://localhost:8080/process");
  });

  test("shows health dot and tooltip content", () => {
    const step = createStep("sync");

    render(<Footer step={step} healthStatus="healthy" />);

    expect(screen.getByTestId("health-dot")).toBeInTheDocument();
    expect(
      screen.getByText(t("overviewStep.healthStatus"))
    ).toBeInTheDocument();
  });

  test("renders flow goals for flow step", () => {
    const step = createStep("flow", { goals: ["goal-a", "goal-b"] });

    const { container } = render(<Footer step={step} healthStatus="healthy" />);

    const endpoint = container.querySelector(".step-endpoint");
    expect(endpoint?.textContent).toBe("goal-a, goal-b");
    expect(screen.getByText(t("stepFooter.flowGoals"))).toBeInTheDocument();
    const tooltipSections = screen.getAllByTestId("tooltip-section");
    const hasGoalsSection = tooltipSections.some((section) =>
      section.textContent?.includes("goal-a, goal-b")
    );
    expect(hasGoalsSection).toBe(true);
  });
});
