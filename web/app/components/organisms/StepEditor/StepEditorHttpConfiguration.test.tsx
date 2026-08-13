import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import StepEditorHttpConfiguration from "./StepEditorHttpConfiguration";

jest.mock("@/app/components/molecules/DurationInput", () => ({
  __esModule: true,
  default: ({ value, onChange }: any) => (
    <input
      data-testid="duration-input"
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
    />
  ),
}));

describe("StepEditorHttpConfiguration", () => {
  const baseProps = {
    endpoint: "http://localhost:8080/test",
    httpMethod: "POST" as const,
    healthCheck: "http://localhost:8080/health",
    compensate: "",
    compensateMethod: "POST" as const,
    compensateTimeout: 0,
    httpTimeout: 5000,
    memoizable: false,
    setEndpoint: jest.fn(),
    setHttpMethod: jest.fn(),
    setHealthCheck: jest.fn(),
    setCompensate: jest.fn(),
    setCompensateMethod: jest.fn(),
    setCompensateTimeout: jest.fn(),
    setHttpTimeout: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders HTTP configuration fields", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    expect(
      screen.getByText(t("stepEditor.httpConfigLabel"))
    ).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("http://localhost:8080/test")
    ).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "POST" })).toHaveLength(2);
    expect(
      screen.getByDisplayValue("http://localhost:8080/health")
    ).toBeInTheDocument();
    const getButton = screen.getByRole("button", { name: "GET" });
    expect(getButton).toBeDisabled();
    expect(screen.getAllByTestId("duration-input")[0]).toHaveValue("5000");
  });

  test("updates method, endpoint, timeout, and health check", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    fireEvent.click(screen.getAllByRole("button", { name: "POST" })[0]);
    fireEvent.click(screen.getByRole("option", { name: "GET" }));
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.endpointPlaceholder")),
      {
        target: { value: "http://localhost:9090/new" },
      }
    );
    fireEvent.change(screen.getAllByTestId("duration-input")[0], {
      target: { value: "10000" },
    });
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.healthCheckPlaceholder")),
      {
        target: { value: "http://localhost:9090/health" },
      }
    );

    expect(baseProps.setHttpMethod).toHaveBeenCalledWith("GET");
    expect(baseProps.setEndpoint).toHaveBeenCalledWith(
      "http://localhost:9090/new"
    );
    expect(baseProps.setHttpTimeout).toHaveBeenCalledWith(10000);
    expect(baseProps.setHealthCheck).toHaveBeenCalledWith(
      "http://localhost:9090/health"
    );
  });

  test("renders compensate field and calls setCompensate on change", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    const compensateInput = screen.getByPlaceholderText(
      t("stepEditor.compensatePlaceholder")
    );
    expect(compensateInput).toBeInTheDocument();

    fireEvent.change(compensateInput, {
      target: { value: "http://localhost:8080/compensate" },
    });

    expect(baseProps.setCompensate).toHaveBeenCalledWith(
      "http://localhost:8080/compensate"
    );
  });

  test("updates compensate method and timeout", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    fireEvent.click(screen.getAllByRole("button", { name: "POST" })[1]);
    fireEvent.click(screen.getByRole("option", { name: "DELETE" }));
    fireEvent.change(screen.getAllByTestId("duration-input")[1], {
      target: { value: "2000" },
    });

    expect(baseProps.setCompensateMethod).toHaveBeenCalledWith("DELETE");
    expect(baseProps.setCompensateTimeout).toHaveBeenCalledWith(2000);
  });

  test("disables compensate field when memoizable is true", () => {
    render(<StepEditorHttpConfiguration {...baseProps} memoizable={true} />);

    const compensateInput = screen.getByPlaceholderText(
      t("stepEditor.compensatePlaceholder")
    );
    expect(compensateInput).toBeDisabled();
  });
});
