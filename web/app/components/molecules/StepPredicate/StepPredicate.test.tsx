import { render, screen } from "@testing-library/react";
import StepPredicate from "./StepPredicate";
import { Step, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import { t } from "@/app/testUtils/i18n";

describe("StepPredicate", () => {
  const createStep = (
    predicateScript?: string,
    predicateLanguage?: string
  ): Step => ({
    id: "step-1",
    name: "Test Step",
    type: "service",
    attributes: {},

    http: {
      endpoint: "http://localhost:8080/test",
      timeout: 5000,
    },
    predicate:
      predicateScript && predicateLanguage
        ? {
            language: predicateLanguage,
            script: predicateScript,
          }
        : undefined,
  });

  test("renders predicate with Lua language", () => {
    const step = createStep("return temperature > 100", SCRIPT_LANGUAGE_LUA);
    const { container } = render(<StepPredicate step={step} />);

    expect(
      screen.getByText((content) =>
        content.startsWith(t("stepPredicate.title", { language: "lua" }))
      )
    ).toBeInTheDocument();
    expect(container.querySelector(".predicate-code")?.textContent).toBe(
      "return temperature > 100"
    );
  });

  test("does not render when predicate is undefined", () => {
    const step = createStep();
    const { container } = render(<StepPredicate step={step} />);

    expect(container.firstChild).toBeNull();
  });

  test("renders complex predicate expression", () => {
    const step = createStep(
      "return temperature > 50 and humidity < 80",
      SCRIPT_LANGUAGE_LUA
    );
    const { container } = render(<StepPredicate step={step} />);

    expect(container.querySelector(".predicate-code")?.textContent).toBe(
      "return temperature > 50 and humidity < 80"
    );
  });
});
