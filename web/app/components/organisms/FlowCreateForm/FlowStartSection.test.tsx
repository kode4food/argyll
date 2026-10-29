import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import FlowStartSection from "./FlowStartSection";

describe("FlowStartSection", () => {
  const baseProps = {
    creating: false,
    disabled: false,
    flowId: "flow-1",
    onCreateFlow: jest.fn(),
    onFlowIdChange: jest.fn(),
    onGenerateId: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders flow id controls", () => {
    render(<FlowStartSection {...baseProps} />);

    expect(
      screen.getByLabelText(t("flowCreate.generateIdAria"))
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: t("common.start") })
    ).toBeInTheDocument();
  });

  test("handles flow id input, generate, and start actions", () => {
    render(<FlowStartSection {...baseProps} />);

    fireEvent.change(
      screen.getByPlaceholderText(t("flowCreate.flowIdPlaceholder")),
      {
        target: { value: "flow-2" },
      }
    );
    fireEvent.click(screen.getByLabelText(t("flowCreate.generateIdAria")));
    fireEvent.click(screen.getByRole("button", { name: t("common.start") }));

    expect(baseProps.onFlowIdChange).toHaveBeenCalledWith("flow-2");
    expect(baseProps.onGenerateId).toHaveBeenCalled();
    expect(baseProps.onCreateFlow).toHaveBeenCalled();
  });

  test("does not offer a compensation opt-out", () => {
    render(<FlowStartSection {...baseProps} />);

    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  });

  test("disables the whole section when disabled", () => {
    render(<FlowStartSection {...baseProps} disabled={true} />);

    expect(
      screen.getByPlaceholderText(t("flowCreate.flowIdPlaceholder"))
    ).toBeDisabled();
    expect(
      screen.getByLabelText(t("flowCreate.generateIdAria"))
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: t("common.start") })
    ).toBeDisabled();
  });

  test("keeps the flow id editable when the id is empty", () => {
    render(<FlowStartSection {...baseProps} flowId="  " />);

    expect(
      screen.getByPlaceholderText(t("flowCreate.flowIdPlaceholder"))
    ).toBeEnabled();
    expect(
      screen.getByRole("button", { name: t("common.start") })
    ).toBeDisabled();
  });
});
