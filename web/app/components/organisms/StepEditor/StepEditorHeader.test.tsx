import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import StepEditorHeader from "./StepEditorHeader";

describe("StepEditorHeader", () => {
  const baseProps = {
    isCreateMode: false,
    stepId: "step-1",
    memoizable: false,
    memoizableDisabled: false,
    onMemoizableChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("toggles memoizable", () => {
    render(<StepEditorHeader {...baseProps} />);

    fireEvent.click(screen.getByRole("checkbox"));

    expect(baseProps.onMemoizableChange).toHaveBeenCalledWith(true);
  });

  test("disables memoizable when a compensate URL is set", () => {
    render(<StepEditorHeader {...baseProps} memoizableDisabled={true} />);

    expect(screen.getByRole("checkbox")).toBeDisabled();
    expect(
      screen.getByTitle(t("stepEditor.memoizableDisabled"))
    ).toBeInTheDocument();
  });
});
