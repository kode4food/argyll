import { act, fireEvent, render, screen } from "@testing-library/react";
import { api } from "@/app/api";
import { useChatStore } from "@/app/store/chatStore";
import { t } from "@/app/testUtils/i18n";
import ChatBox from "./ChatBox";

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: { chatWithAgent: jest.fn() },
}));

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({
    spaceId: "risk",
    goalSteps: [["notify"], []],
    flowInit: { order_id: ["o-1"] },
  }),
}));

jest.mock("@/app/store/flowStore", () => ({
  useSelectedFlow: () => "flow-1",
}));

jest.mock("@/app/hooks/useAgentCapabilities", () => ({
  useAgentCapabilities: () => ({
    providers: [
      { provider: "anthropic", models: [{ id: "claude" }], mcp: [] },
      { provider: "openai", models: [{ id: "gpt" }], mcp: [] },
    ],
  }),
}));

const chatWithAgent = api.chatWithAgent as jest.MockedFunction<
  typeof api.chatWithAgent
>;

describe("ChatBox", () => {
  beforeEach(() => {
    chatWithAgent.mockReset();
    useChatStore.setState({
      provider: "",
      model: "",
      isOpen: true,
      isPending: false,
      messages: [],
      error: null,
    });
  });

  test("disables the composer until a model is selected", () => {
    render(<ChatBox />);
    expect(screen.getByLabelText(t("chat.placeholder"))).toBeDisabled();

    fireEvent.click(screen.getByLabelText(t("chat.provider")));
    fireEvent.click(screen.getByText("anthropic"));
    fireEvent.click(screen.getByLabelText(t("chat.model")));
    fireEvent.click(screen.getByText("claude"));

    expect(useChatStore.getState()).toMatchObject({
      provider: "anthropic",
      model: "claude",
    });
    expect(screen.getByLabelText(t("chat.placeholder"))).toBeEnabled();
  });

  test("sends a message and shows the reply", async () => {
    useChatStore.setState({ provider: "anthropic", model: "claude" });
    chatWithAgent.mockResolvedValue({ reply: "hello", usage: {} });
    render(<ChatBox />);

    const input = screen.getByLabelText(t("chat.placeholder"));
    fireEvent.change(input, { target: { value: " hi " } });
    await act(async () => {
      fireEvent.keyDown(input, { key: "Enter" });
    });

    expect(chatWithAgent).toHaveBeenCalledWith({
      provider: "anthropic",
      model: "claude",
      messages: [{ role: "user", content: "hi" }],
      view: {
        space_id: "risk",
        flow_id: "flow-1",
        goals: { steps: ["notify"] },
        init: { order_id: ["o-1"] },
        locale: "en-US",
      },
    });
    expect(await screen.findByText("hello")).toBeInTheDocument();
    expect(input).toHaveValue("");
  });

  test("closes", () => {
    render(<ChatBox />);
    fireEvent.click(screen.getByLabelText(t("chat.close")));
    expect(useChatStore.getState().isOpen).toBe(false);
  });
});
