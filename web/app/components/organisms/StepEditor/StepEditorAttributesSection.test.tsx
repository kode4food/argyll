import { fireEvent, render, screen } from "@testing-library/react";
import { AttributeType } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorAttributesSection from "./StepEditorAttributesSection";

describe("StepEditorAttributesSection", () => {
  const attributes = [
    {
      id: "request",
      role: "required" as const,
      name: "request",
      dataType: AttributeType.String,
    },
    {
      id: "result",
      role: "output" as const,
      name: "result",
      dataType: AttributeType.String,
    },
  ];
  const baseProps = {
    addAttribute: jest.fn(),
    attributes,
    flowInputOptions: [],
    flowOutputOptions: [],
    removeAttribute: jest.fn(),
    stepType: "service" as const,
    handling: "standard" as const,
    updateAttribute: jest.fn(),
  };

  beforeEach(() => jest.clearAllMocks());

  test("shows compensation toggles", () => {
    const { rerender } = render(<StepEditorAttributesSection {...baseProps} />);

    expect(
      screen.queryByRole("button", {
        name: `${t("stepEditor.compensatedAttributeLabel")} request`,
      })
    ).not.toBeInTheDocument();

    rerender(
      <StepEditorAttributesSection {...baseProps} handling="compensated" />
    );
    const compensateButton = screen.getByRole("button", {
      name: `${t("stepEditor.compensatedAttributeLabel")} request`,
    });
    const roleButton = screen.getAllByRole("button", {
      name: t("stepEditor.attrTypeSelect"),
    })[0];

    expect(compensateButton.parentElement).toBe(
      roleButton.parentElement?.parentElement
    );
    fireEvent.click(compensateButton);

    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "request",
      "compensated",
      true
    );
  });

  test("blocks a duplicate selected inner name", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        handling="compensated"
        attributes={[
          { ...attributes[0], mappingName: "value", compensated: true },
          { ...attributes[1], mappingName: "value" },
        ]}
      />
    );

    expect(
      screen.getByRole("button", {
        name: `${t("stepEditor.compensatedAttributeLabel")} result`,
      })
    ).toBeDisabled();
  });
});
