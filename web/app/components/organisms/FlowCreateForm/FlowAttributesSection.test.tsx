import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import FlowAttributesSection from "./FlowAttributesSection";

jest.mock("@/app/components/molecules/LazyCodeEditor", () => {
  return function MockLazyCodeEditor({ value, onChange }: any) {
    return (
      <textarea
        data-testid="code-editor"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  };
});

describe("FlowAttributesSection", () => {
  const baseProps = {
    editorMode: "basic" as const,
    emptyAttributesLabel: "No attributes",
    flowInputOptions: [],
    flowInputValues: {},
    flowInputValuesRaw: {},
    getFlowInputPlaceholder: () => "",
    handleBasicInputChange: jest.fn(),
    initialState: "{}",
    jsonError: null,
    onEditorModeChange: jest.fn(),
    onFocusedPreviewAttributeChange: jest.fn(),
    setInitialState: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders empty basic-mode state", () => {
    render(<FlowAttributesSection {...baseProps} />);

    expect(
      screen.getByText(t("flowCreate.requiredAttributesLabel"))
    ).toBeInTheDocument();
    expect(screen.getByText("No attributes")).toBeInTheDocument();
  });

  test("switches editor modes through callbacks", () => {
    render(<FlowAttributesSection {...baseProps} />);

    fireEvent.click(
      screen.getByRole("button", { name: t("flowCreate.modeJson") })
    );
    expect(baseProps.onEditorModeChange).toHaveBeenCalledWith("json");
  });

  test("renders JSON editor and error state", () => {
    render(
      <FlowAttributesSection
        {...baseProps}
        editorMode="json"
        jsonError="bad json"
      />
    );

    expect(screen.getByTestId("code-editor")).toBeInTheDocument();
    expect(
      screen.getByText((content) =>
        content.startsWith(t("flowCreate.invalidJson", { error: "" }))
      )
    ).toBeInTheDocument();
  });

  test("uses an exclamation-circle icon for missing required inputs", () => {
    render(
      <FlowAttributesSection
        {...baseProps}
        flowInputOptions={[
          {
            name: "order_id",
            required: true,
          },
        ]}
      />
    );

    const badge = screen.getByLabelText(t("flowCreate.badgeRequiredMissing"));
    const svg = badge.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg?.getAttribute("class")).toContain("lucide-circle-alert");
    expect(svg?.getAttribute("class")).not.toContain("lucide-check-circle-2");
  });

  test("treats hidden raw values as satisfied in basic mode", () => {
    render(
      <FlowAttributesSection
        {...baseProps}
        flowInputOptions={[
          {
            name: "quantity",
            required: true,
            defaultValue: "0",
          },
        ]}
        flowInputValues={{ quantity: "" }}
        flowInputValuesRaw={{ quantity: "0" }}
      />
    );

    const badge = screen.getByLabelText(t("flowCreate.badgeRequiredSatisfied"));
    const svg = badge.querySelector("svg");
    expect(svg).toBeInTheDocument();
    expect(svg?.getAttribute("class")).toContain("lucide-circle-check");
  });
});
