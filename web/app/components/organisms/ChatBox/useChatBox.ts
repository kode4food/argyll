import { useState } from "react";
import { useAgentCapabilities } from "@/app/hooks/useAgentCapabilities";
import { useChatStore } from "@/app/store/chatStore";

export const useChatBox = () => {
  const { providers } = useAgentCapabilities();
  const chat = useChatStore();
  const [draft, setDraft] = useState("");

  const models =
    providers
      .find((item) => item.provider === chat.provider)
      ?.models.map(({ id }) => id) ?? [];
  const isReady = chat.provider !== "" && chat.model !== "";

  const handleSubmit = () => {
    const content = draft.trim();
    setDraft("");
    void chat.send(content);
  };

  return {
    providers: providers.map((item) => item.provider),
    models,
    provider: chat.provider,
    model: chat.model,
    messages: chat.messages,
    isPending: chat.isPending,
    error: chat.error,
    isComposerDisabled: !isReady || chat.isPending,
    draft,
    setDraft,
    setProvider: chat.setProvider,
    setModel: chat.setModel,
    close: chat.toggleOpen,
    handleSubmit,
  };
};
