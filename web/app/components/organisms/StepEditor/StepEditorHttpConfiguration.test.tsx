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
    handling: "standard" as const,
    stepType: "sync" as const,
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

  test("updates invoke configuration", () => {
    render(<StepEditorHttpConfiguration {...baseProps} />);

    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    fireEvent.click(screen.getByRole("option", { name: "GET" }));
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.endpointPlaceholder")),
      { target: { value: "http://localhost:9090/new" } }
    );
    fireEvent.change(screen.getByTestId("duration-input"), {
      target: { value: "10000" },
    });
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.healthCheckPlaceholder")),
      { target: { value: "http://localhost:9090/health" } }
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

  test("shows compensated configuration", () => {
    const { rerender } = render(<StepEditorHttpConfiguration {...baseProps} />);

    expect(
      screen.queryByPlaceholderText(t("stepEditor.compensatePlaceholder"))
    ).not.toBeInTheDocument();

    rerender(
      <StepEditorHttpConfiguration {...baseProps} handling="compensated" />
    );

    expect(
      screen.getByPlaceholderText(t("stepEditor.compensatePlaceholder"))
    ).toBeInTheDocument();
  });

  test("updates compensation configuration", () => {
    render(
      <StepEditorHttpConfiguration {...baseProps} handling="compensated" />
    );

    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.compensatePlaceholder")),
      { target: { value: "http://localhost:8080/compensate" } }
    );
    fireEvent.click(screen.getAllByRole("button", { name: "POST" })[1]);
    fireEvent.click(screen.getByRole("option", { name: "DELETE" }));
    fireEvent.change(screen.getAllByTestId("duration-input")[1], {
      target: { value: "2000" },
    });

    expect(baseProps.setCompensate).toHaveBeenCalledWith(
      "http://localhost:8080/compensate"
    );
    expect(baseProps.setCompensateMethod).toHaveBeenCalledWith("DELETE");
    expect(baseProps.setCompensateTimeout).toHaveBeenCalledWith(2000);
  });
});
