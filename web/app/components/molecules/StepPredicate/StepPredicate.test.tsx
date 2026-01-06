import { render, screen } from "@testing-library/react";
import StepPredicate from "./StepPredicate";
import { Step, SCRIPT_LANGUAGE_ALE, SCRIPT_LANGUAGE_LUA } from "@/app/api";

describe("StepPredicate", () => {
  const createStep = (
    predicateScript?: string,
    predicateLanguage?: string
  ): Step => ({
    id: "step-1",
    name: "Test Step",
    type: "sync",
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

  test("renders predicate with Ale language", () => {
    const step = createStep("(> temperature 100)", SCRIPT_LANGUAGE_ALE);
    const { container } = render(<StepPredicate step={step} />);

    expect(screen.getByText(/Predicate \(ale\)/i)).toBeInTheDocument();
    expect(container.querySelector(".predicate-code")?.textContent).toBe(
      "(> temperature 100)"
    );
  });

  test("renders predicate with Lua language", () => {
    const step = createStep("return temperature > 100", SCRIPT_LANGUAGE_LUA);
    const { container } = render(<StepPredicate step={step} />);

    expect(screen.getByText(/Predicate \(lua\)/i)).toBeInTheDocument();
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
      "(and (> temperature 50) (< humidity 80))",
      SCRIPT_LANGUAGE_ALE
    );
    const { container } = render(<StepPredicate step={step} />);

    expect(container.querySelector(".predicate-code")?.textContent).toBe(
      "(and (> temperature 50) (< humidity 80))"
    );
  });
});
