import { fireEvent, render, screen } from "@testing-library/react";
import { StepType } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorBasicFields from "./StepEditorBasicFields";

describe("StepEditorBasicFields", () => {
  const renderComponent = (
    stepType: StepType = "service",
    isCreateMode = true
  ) => {
    const props = {
      handling: "standard" as const,
      isCreateMode,
      name: "Example Step",
      setHandling: jest.fn(),
      setName: jest.fn(),
      setStepId: jest.fn(),
      setStepType: jest.fn(),
      setTags: jest.fn(),
      stepId: "step-1",
      stepType,
      tags: [],
      tagVocabulary: [],
    };

    render(<StepEditorBasicFields {...props} />);
    return props;
  };

  test("renders create fields", () => {
    renderComponent();

    expect(
      screen.getByPlaceholderText(t("stepEditor.stepIdPlaceholder"))
    ).toBeEnabled();
    expect(
      screen.getByPlaceholderText(t("stepEditor.stepNamePlaceholder"))
    ).toHaveValue("Example Step");
  });

  test("disables step id in edit mode", () => {
    renderComponent("service", false);

    expect(
      screen.getByPlaceholderText(t("stepEditor.stepIdPlaceholder"))
    ).toBeDisabled();
  });

  test("updates text fields and selected type", () => {
    const props = renderComponent("service");

    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.stepIdPlaceholder")),
      {
        target: { value: "step-2" },
      }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.stepNamePlaceholder")),
      {
        target: { value: "Changed Step" },
      }
    );

    // open the type dropdown, then select a type
    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.typeServiceLabel") })
    );
    fireEvent.click(screen.getByTitle(t("stepEditor.typeFlowTitle")));

    expect(props.setStepId).toHaveBeenCalledWith("step-2");
    expect(props.setName).toHaveBeenCalledWith("Changed Step");
    expect(props.setStepType).toHaveBeenCalledWith("flow");
  });

  test("selects the current type", () => {
    renderComponent("service");

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.typeServiceLabel") })
    );
    expect(screen.getByTitle(t("stepEditor.typeServiceTitle"))).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("changes handling", () => {
    const props = renderComponent();

    expect(document.querySelector(".lucide-play")).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", {
        name: t("stepEditor.handling.standard"),
      })
    );
    fireEvent.click(
      screen.getByRole("option", {
        name: t("stepEditor.handling.compensated"),
      })
    );

    expect(props.setHandling).toHaveBeenCalledWith("compensated");
  });

  test("disables compensation for non-HTTP steps", () => {
    renderComponent("flow");

    fireEvent.click(
      screen.getByRole("button", {
        name: t("stepEditor.handling.standard"),
      })
    );

    expect(
      screen.getByRole("option", {
        name: t("stepEditor.handling.compensated"),
      })
    ).toBeDisabled();
  });

  test("moves dropdown highlight", () => {
    renderComponent("service");

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.typeServiceLabel") })
    );

    const serviceType = screen.getByTitle(t("stepEditor.typeServiceTitle"));
    const flowType = screen.getByTitle(t("stepEditor.typeFlowTitle"));
    expect(serviceType.className).toContain("itemHighlighted");

    fireEvent.mouseEnter(flowType);

    expect(flowType.className).toContain("itemHighlighted");
    expect(serviceType.className).not.toContain("itemHighlighted");
  });
});
