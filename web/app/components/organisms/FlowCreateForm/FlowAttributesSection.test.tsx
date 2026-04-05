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
    statusClassByType: {
      provided: "provided",
      defaulted: "defaulted",
      required: "required",
      optional: "optional",
    },
    statusLabelByType: {
      provided: "Provided",
      defaulted: "Defaulted",
      required: "Required",
      optional: "Optional",
    },
    toFlowInputStatus: jest.fn(() => "required" as const),
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
});
