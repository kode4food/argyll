import { create } from "zustand";
import { devtools, persist, createJSONStorage } from "zustand/middleware";
import {
  api,
  type ChatAction,
  type ChatMessage,
  type ChatView,
} from "@/app/api";

interface ChatState {
  provider: string;
  model: string;
  isOpen: boolean;
  isPending: boolean;
  messages: ChatMessage[];
  error: string | null;
  action: ChatAction | null;
  setProvider: (provider: string) => void;
  setModel: (model: string) => void;
  toggleOpen: () => void;
  send: (content: string, view?: ChatView) => Promise<void>;
  clearAction: () => void;
}

const useChatStore = create<ChatState>()(
  devtools(
    persist(
      (set, get) => ({
        provider: "",
        model: "",
        isOpen: false,
        isPending: false,
        messages: [],
        error: null,
        action: null,
        setProvider: (provider) =>
          set(
            (state) =>
              provider === state.provider ? state : { provider, model: "" },
            false,
            "chat/setProvider"
          ),
        setModel: (model) => set({ model }, false, "chat/setModel"),
        toggleOpen: () =>
          set((state) => ({ isOpen: !state.isOpen }), false, "chat/toggle"),
        send: async (content, view) => {
          const { provider, model, messages } = get();
          const sent: ChatMessage[] = [...messages, { role: "user", content }];
          set(
            { messages: sent, isPending: true, error: null, action: null },
            false,
            "chat/send"
          );
          try {
            const { reply, action } = await api.chatWithAgent({
              provider,
              model,
              messages: sent,
              view,
            });
            set(
              {
                messages: [...sent, { role: "assistant", content: reply }],
                isPending: false,
                action: action ?? null,
              },
              false,
              "chat/reply"
            );
          } catch (error) {
            const message =
              error instanceof Error ? error.message : String(error);
            set(
              { messages, isPending: false, error: message },
              false,
              "chat/failed"
            );
          }
        },
        clearAction: () => set({ action: null }, false, "chat/clearAction"),
      }),
      {
        name: "chatStore",
        storage: createJSONStorage(() => localStorage),
        partialize: ({ provider, model }) => ({ provider, model }),
      }
    ),
    { name: "chatStore" }
  )
);

const useChatOpen = () => useChatStore((state) => state.isOpen);
const useToggleChat = () => useChatStore((state) => state.toggleOpen);

export { useChatStore, useChatOpen, useToggleChat };
