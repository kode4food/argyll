import { render, screen } from "@testing-library/react";
import { SCRIPT_LANGUAGE_ARGYLL, SCRIPT_LANGUAGE_LUA } from "@/app/api";
import ScriptLanguageLogo from "./ScriptLanguageLogo";

describe("ScriptLanguageLogo", () => {
  test("renders Argyll Expressions logo with name as alt", () => {
    render(
      <ScriptLanguageLogo language={SCRIPT_LANGUAGE_ARGYLL} name="Argyll" />
    );
    expect(screen.getByAltText("Argyll")).toHaveAttribute(
      "src",
      "/icons/argex-logo.svg"
    );
  });

  test("renders Lua logo with name as alt", () => {
    render(<ScriptLanguageLogo language={SCRIPT_LANGUAGE_LUA} name="Lua" />);
    expect(screen.getByAltText("Lua")).toHaveAttribute(
      "src",
      "/icons/lua-logo.svg"
    );
  });

  test("shows name beside logo when requested", () => {
    render(
      <ScriptLanguageLogo language={SCRIPT_LANGUAGE_LUA} name="Lua" showName />
    );
    expect(screen.getByText("Lua")).toBeInTheDocument();
    expect(screen.getByRole("presentation")).toHaveAttribute("alt", "");
  });

  test("falls back to name for unknown language", () => {
    render(<ScriptLanguageLogo language="other" name="Other" />);
    expect(screen.getByText("Other")).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
});
