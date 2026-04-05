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
    healthCheck: "http://localhost:8080/health",
    httpTimeout: 5000,
    setEndpoint: jest.fn(),
    setHealthCheck: jest.fn(),
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
    expect(
      screen.getByDisplayValue("http://localhost:8080/health")
    ).toBeInTheDocument();
    expect(screen.getByTestId("duration-input")).toHaveValue("5000");
  });

  test("updates endpoint, timeout, and health check", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

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

    expect(baseProps.setEndpoint).toHaveBeenCalledWith(
      "http://localhost:9090/new"
    );
    expect(baseProps.setHttpTimeout).toHaveBeenCalledWith(10000);
    expect(baseProps.setHealthCheck).toHaveBeenCalledWith(
      "http://localhost:9090/health"
    );
  });
});
