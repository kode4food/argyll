import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import FlowStartSection from "./FlowStartSection";

describe("FlowStartSection", () => {
  const baseProps = {
    creating: false,
    disableStart: false,
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
});
