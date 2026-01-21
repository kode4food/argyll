import { render, screen, fireEvent } from "@testing-library/react";
import ScriptConfigEditor from "./ScriptConfigEditor";
import { SCRIPT_LANGUAGE_ALE, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/components/molecules/ScriptEditor", () => {
  return function MockScriptEditor({
    value,
    onChange,
    language,
    readOnly,
  }: any) {
    return (
      <div data-testid="script-editor">
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          readOnly={readOnly}
          data-language={language}
        />
      </div>
    );
  };
});

describe("ScriptConfigEditor", () => {
  const defaultProps = {
    label: "Test Script",
    value: "test code",
    onChange: jest.fn(),
    language: SCRIPT_LANGUAGE_ALE,
    onLanguageChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders label", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    expect(screen.getByText("Test Script")).toBeInTheDocument();
  });

  test("renders language buttons when not readOnly", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    expect(screen.getByText(t("script.language.ale"))).toBeInTheDocument();
    expect(screen.getByText(t("script.language.lua"))).toBeInTheDocument();
  });

  test("does not render language buttons when readOnly", () => {
    render(<ScriptConfigEditor {...defaultProps} readOnly />);
    expect(
      screen.queryByText(t("script.language.ale"))
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(t("script.language.lua"))
    ).not.toBeInTheDocument();
  });

  test("calls onLanguageChange with Ale when Ale button clicked", () => {
    render(
      <ScriptConfigEditor {...defaultProps} language={SCRIPT_LANGUAGE_LUA} />
    );
    const aleButton = screen.getByText(t("script.language.ale"));
    fireEvent.click(aleButton);
    expect(defaultProps.onLanguageChange).toHaveBeenCalledWith(
      SCRIPT_LANGUAGE_ALE
    );
  });

  test("calls onLanguageChange with Lua when Lua button clicked", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    const luaButton = screen.getByText(t("script.language.lua"));
    fireEvent.click(luaButton);
    expect(defaultProps.onLanguageChange).toHaveBeenCalledWith(
      SCRIPT_LANGUAGE_LUA
    );
  });

  test("marks Ale button as active when language is Ale", () => {
    render(
      <ScriptConfigEditor {...defaultProps} language={SCRIPT_LANGUAGE_ALE} />
    );
    const aleButton = screen.getByText(t("script.language.ale"));
    expect(aleButton.className).toContain("languageButtonActive");
  });

  test("marks Lua button as active when language is Lua", () => {
    render(
      <ScriptConfigEditor {...defaultProps} language={SCRIPT_LANGUAGE_LUA} />
    );
    const luaButton = screen.getByText(t("script.language.lua"));
    expect(luaButton.className).toContain("languageButtonActive");
  });

  test("passes value to ScriptEditor", () => {
    render(<ScriptConfigEditor {...defaultProps} value="custom code" />);
    const textarea = screen.getByRole("textbox");
    expect(textarea).toHaveValue("custom code");
  });

  test("passes onChange to ScriptEditor", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    const textarea = screen.getByRole("textbox");
    fireEvent.change(textarea, { target: { value: "new code" } });
    expect(defaultProps.onChange).toHaveBeenCalledWith("new code");
  });

  test("passes language to ScriptEditor", () => {
    render(
      <ScriptConfigEditor {...defaultProps} language={SCRIPT_LANGUAGE_LUA} />
    );
    const textarea = screen.getByRole("textbox");
    expect(textarea).toHaveAttribute("data-language", SCRIPT_LANGUAGE_LUA);
  });

  test("passes readOnly to ScriptEditor", () => {
    render(<ScriptConfigEditor {...defaultProps} readOnly />);
    const textarea = screen.getByRole("textbox");
    expect(textarea).toHaveAttribute("readOnly");
  });

  test("uses default containerClassName when not provided", () => {
    const { container } = render(<ScriptConfigEditor {...defaultProps} />);
    expect(
      container.querySelector(".scriptEditorContainer")
    ).toBeInTheDocument();
  });

  test("uses custom containerClassName when provided", () => {
    const { container } = render(
      <ScriptConfigEditor {...defaultProps} containerClassName="customClass" />
    );
    expect(container.querySelector(".customClass")).toBeInTheDocument();
  });

  test("blurs button after Ale click", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    const aleButton = screen.getByText("Ale") as HTMLButtonElement;
    const blurSpy = jest.spyOn(aleButton, "blur");

    fireEvent.click(aleButton);

    expect(blurSpy).toHaveBeenCalled();
    blurSpy.mockRestore();
  });

  test("blurs button after Lua click", () => {
    render(<ScriptConfigEditor {...defaultProps} />);
    const luaButton = screen.getByText("Lua") as HTMLButtonElement;
    const blurSpy = jest.spyOn(luaButton, "blur");

    fireEvent.click(luaButton);

    expect(blurSpy).toHaveBeenCalled();
    blurSpy.mockRestore();
  });
});
