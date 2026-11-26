import { api } from "@/app/api";
import { useChatStore } from "./chatStore";

jest.mock("@/app/api", () => ({
  api: { chatWithAgent: jest.fn() },
}));

const chatWithAgent = api.chatWithAgent as jest.MockedFunction<
  typeof api.chatWithAgent
>;

describe("chatStore", () => {
  beforeEach(() => {
    localStorage.clear();
    chatWithAgent.mockReset();
    useChatStore.setState({
      provider: "",
      model: "",
      isOpen: false,
      isPending: false,
      messages: [],
      error: null,
      action: null,
    });
  });

  test("setProvider clears the model on change", () => {
    useChatStore.getState().setProvider("anthropic");
    useChatStore.getState().setModel("claude");
    useChatStore.getState().setProvider("anthropic");
    expect(useChatStore.getState().model).toBe("claude");

    useChatStore.getState().setProvider("openai");
    expect(useChatStore.getState()).toMatchObject({
      provider: "openai",
      model: "",
    });
  });

  test("toggleOpen flips visibility", () => {
    useChatStore.getState().toggleOpen();
    expect(useChatStore.getState().isOpen).toBe(true);
    useChatStore.getState().toggleOpen();
    expect(useChatStore.getState().isOpen).toBe(false);
  });

  test("persists only the selection", () => {
    useChatStore.getState().setProvider("anthropic");
    useChatStore.getState().setModel("claude");
    useChatStore.getState().toggleOpen();

    const stored = JSON.parse(localStorage.getItem("chatStore") ?? "{}");
    expect(stored.state).toEqual({ provider: "anthropic", model: "claude" });
  });

  test("send appends the reply", async () => {
    useChatStore.setState({ provider: "anthropic", model: "claude" });
    chatWithAgent.mockResolvedValue({ reply: "hello", usage: {} });

    await useChatStore.getState().send("hi");

    expect(chatWithAgent).toHaveBeenCalledWith({
      provider: "anthropic",
      model: "claude",
      messages: [{ role: "user", content: "hi" }],
    });
    expect(useChatStore.getState()).toMatchObject({
      isPending: false,
      error: null,
      messages: [
        { role: "user", content: "hi" },
        { role: "assistant", content: "hello" },
      ],
    });
  });

  test("send keeps the reply's action until it is cleared", async () => {
    const action = { type: "open_flow", flow_id: "f1" } as const;
    chatWithAgent.mockResolvedValue({ reply: "started", action, usage: {} });

    await useChatStore.getState().send("start it");
    expect(useChatStore.getState().action).toEqual(action);

    useChatStore.getState().clearAction();
    expect(useChatStore.getState().action).toBeNull();
  });

  test("send drops an action the next message makes stale", async () => {
    useChatStore.setState({ action: { type: "open_space" } });
    chatWithAgent.mockResolvedValue({ reply: "ok", usage: {} });

    const sent = useChatStore.getState().send("never mind");
    expect(useChatStore.getState().action).toBeNull();
    await sent;
    expect(useChatStore.getState().action).toBeNull();
  });

  test("send failure restores the conversation", async () => {
    chatWithAgent.mockRejectedValue(new Error("no key"));

    await useChatStore.getState().send("hi");

    expect(useChatStore.getState()).toMatchObject({
      isPending: false,
      error: "no key",
      messages: [],
    });
  });
});
