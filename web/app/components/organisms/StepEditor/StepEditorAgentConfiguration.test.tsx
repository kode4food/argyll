import { fireEvent, render, screen, within } from "@testing-library/react";
import type { AgentConfig } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import StepEditorAgentConfiguration from "./StepEditorAgentConfiguration";

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

jest.mock("@/app/components/molecules/ScriptEditor", () => ({
  __esModule: true,
  default: ({ value, onChange, language }: any) => (
    <textarea
      data-testid="prologue-editor"
      data-language={language}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock("@/app/components/molecules/TagInput", () => ({
  __esModule: true,
  default: ({ tags, suggestions, shouldEnforceSuggestions }: any) => (
    <div
      data-testid="tool-tags"
      data-suggestions={suggestions.join(",")}
      data-enforced={String(!!shouldEnforceSuggestions)}
    >
      {tags.join(",")}
    </div>
  ),
}));

jest.mock("./useAgentCapabilities", () => ({
  useAgentCapabilities: () => [
    {
      provider: "anthropic",
      models: ["claude-sonnet-5-5", "claude-haiku"],
      mcp: [
        { server: "crm", tools: ["delete", "lookup"] },
        { server: "docs", tools: ["search"] },
      ],
    },
    { provider: "openai", models: ["gpt"], mcp: [] },
  ],
}));

const AGENT: AgentConfig = {
  provider: "anthropic",
  model: "claude-sonnet-5-5",
  prologue: "# Score",
  max_turns: 4,
  max_tokens: 1024,
  mcp: [{ server: "crm", tools: ["lookup"] }],
};

describe("StepEditorAgentConfiguration", () => {
  const renderConfig = (agent: AgentConfig = AGENT) => {
    const setAgent = jest.fn();
    render(<StepEditorAgentConfiguration agent={agent} setAgent={setAgent} />);
    return setAgent;
  };

  test("renders the configuration with a Markdown prologue", () => {
    renderConfig();

    const config = within(
      screen.getByRole("region", { name: t("stepEditor.agentConfigLabel") })
    );
    expect(config.getByText("anthropic")).toBeInTheDocument();
    expect(config.getByText("claude-sonnet-5-5")).toBeInTheDocument();
    expect(config.getByTestId("prologue-editor")).toHaveAttribute(
      "data-language",
      "markdown"
    );
    expect(config.getByText("crm")).toBeInTheDocument();
    const tools = config.getByTestId("tool-tags");
    expect(tools).toHaveTextContent("lookup");
    expect(tools).toHaveAttribute("data-suggestions", "delete,lookup");
    expect(tools).toHaveAttribute("data-enforced", "true");
  });

  test("chooses an MCP server the worker offers", () => {
    const setAgent = renderConfig();

    fireEvent.click(screen.getByText("crm"));
    fireEvent.click(screen.getByText("docs"));
    expect(setAgent).toHaveBeenLastCalledWith({
      ...AGENT,
      mcp: [{ server: "docs", tools: undefined }],
    });
  });

  test("keeps a server the worker no longer offers", () => {
    renderConfig({ ...AGENT, mcp: [{ server: "legacy" }] });

    expect(screen.getByText("legacy")).toBeInTheDocument();
    expect(screen.getByTestId("tool-tags")).toHaveAttribute(
      "data-enforced",
      "false"
    );
  });

  test("selects providers and models from the catalog", () => {
    const setAgent = renderConfig();

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.agentModelLabel") })
    );
    fireEvent.click(screen.getByText("claude-haiku"));
    expect(setAgent).toHaveBeenLastCalledWith({
      ...AGENT,
      model: "claude-haiku",
    });

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.agentProviderLabel") })
    );
    fireEvent.click(screen.getByText("openai"));
    expect(setAgent).toHaveBeenLastCalledWith({
      ...AGENT,
      provider: "openai",
      model: "",
      mcp: [],
    });
  });

  test("disables MCP selection and adding when there are no servers", () => {
    renderConfig({ ...AGENT, provider: "openai", mcp: [{ server: "legacy" }] });

    const selector = screen.getByRole("button", {
      name: t("stepEditor.agentMCPLabel"),
    });
    expect(selector).toBeDisabled();
    expect(selector).toHaveTextContent("legacy");
    expect(screen.getByTitle(t("stepEditor.addMCPServer"))).toBeDisabled();
    fireEvent.click(selector);
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  test("enables adding MCP servers once a provider with servers is selected", () => {
    const setAgent = jest.fn();
    const empty = { ...AGENT, provider: "", model: "", mcp: [] };
    const { rerender } = render(
      <StepEditorAgentConfiguration agent={empty} setAgent={setAgent} />
    );
    const add = screen.getByTitle(t("stepEditor.addMCPServer"));
    expect(add).toBeDisabled();
    fireEvent.click(add);
    expect(setAgent).not.toHaveBeenCalled();

    rerender(
      <StepEditorAgentConfiguration
        agent={{ ...empty, provider: AGENT.provider }}
        setAgent={setAgent}
      />
    );
    expect(add).toBeEnabled();
  });

  test("updates fields", () => {
    const setAgent = renderConfig();

    fireEvent.change(screen.getByDisplayValue("4"), {
      target: { value: "6" },
    });
    expect(setAgent).toHaveBeenLastCalledWith({ ...AGENT, max_turns: 6 });

    fireEvent.change(screen.getByTestId("prologue-editor"), {
      target: { value: "# Rank" },
    });
    expect(setAgent).toHaveBeenLastCalledWith({ ...AGENT, prologue: "# Rank" });
  });

  test("adds and removes MCP servers", () => {
    const setAgent = renderConfig();

    fireEvent.click(screen.getByTitle(t("stepEditor.addMCPServer")));
    expect(setAgent).toHaveBeenLastCalledWith({
      ...AGENT,
      mcp: [...AGENT.mcp!, { server: "" }],
    });

    fireEvent.click(screen.getByTitle(t("stepEditor.removeMCPServer")));
    expect(setAgent).toHaveBeenLastCalledWith({ ...AGENT, mcp: [] });
  });
});
