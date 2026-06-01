import { fireEvent, render, screen } from "@testing-library/react";
import { StepType } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorBasicFields from "./StepEditorBasicFields";

describe("StepEditorBasicFields", () => {
  const renderComponent = (
    stepType: StepType = "sync",
    isCreateMode = true
  ) => {
    const props = {
      isCreateMode,
      name: "Example Step",
      setName: jest.fn(),
      setStepId: jest.fn(),
      setStepType: jest.fn(),
      stepId: "step-1",
      stepType,
    };

    render(<StepEditorBasicFields {...props} />);
    return props;
  };

  test("renders editable identity fields in create mode", () => {
    renderComponent();

    expect(
      screen.getByPlaceholderText(t("stepEditor.stepIdPlaceholder"))
    ).toBeEnabled();
    expect(
      screen.getByPlaceholderText(t("stepEditor.stepNamePlaceholder"))
    ).toHaveValue("Example Step");
  });

  test("disables step id in edit mode", () => {
    renderComponent("sync", false);

    expect(
      screen.getByPlaceholderText(t("stepEditor.stepIdPlaceholder"))
    ).toBeDisabled();
  });

  test("updates text fields and selected type", () => {
    const props = renderComponent("sync");

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
      screen.getByRole("button", { name: t("stepEditor.typeSyncLabel") })
    );
    fireEvent.click(screen.getByTitle(t("stepEditor.typeFlowTitle")));

    expect(props.setStepId).toHaveBeenCalledWith("step-2");
    expect(props.setName).toHaveBeenCalledWith("Changed Step");
    expect(props.setStepType).toHaveBeenCalledWith("flow");
  });

  test("marks the current type as selected in the dropdown", () => {
    renderComponent("async");

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.typeAsyncLabel") })
    );
    expect(screen.getByTitle(t("stepEditor.typeAsyncTitle"))).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });
});
