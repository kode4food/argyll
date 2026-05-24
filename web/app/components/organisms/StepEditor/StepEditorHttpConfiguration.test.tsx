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
    httpTimeout: 5000,
    memoizable: false,
    setEndpoint: jest.fn(),
    setHttpMethod: jest.fn(),
    setHealthCheck: jest.fn(),
    setCompensate: jest.fn(),
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
    expect(screen.getByRole("button", { name: "POST" })).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("http://localhost:8080/health")
    ).toBeInTheDocument();
    expect(screen.getByTestId("duration-input")).toHaveValue("5000");
  });

  test("updates method, endpoint, timeout, and health check", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    fireEvent.click(screen.getByRole("option", { name: "GET" }));
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.endpointPlaceholder")),
      {
        target: { value: "http://localhost:9090/new" },
      }
    );
    fireEvent.change(screen.getByTestId("duration-input"), {
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

  test("disables compensate field when memoizable is true", () => {
    render(<StepEditorHttpConfiguration {...baseProps} memoizable={true} />);

    const compensateInput = screen.getByPlaceholderText(
      t("stepEditor.compensatePlaceholder")
    );
    expect(compensateInput).toBeDisabled();
  });
});
