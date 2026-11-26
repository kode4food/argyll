import { useState } from "react";
import { ChatView, fromGoalSets } from "@/app/api";
import { useUI } from "@/app/contexts/UIContext";
import { useAgentCapabilities } from "@/app/hooks/useAgentCapabilities";
import { useLocale } from "@/app/i18n";
import { useChatStore } from "@/app/store/chatStore";
import { useSelectedFlow } from "@/app/store/flowStore";
import { nonEmptyGoalSets } from "@/utils/goalSets";

export const useChatBox = () => {
  const { providers } = useAgentCapabilities();
  const chat = useChatStore();
  const { spaceId, goalSteps, flowInit } = useUI();
  const flowId = useSelectedFlow();
  const locale = useLocale();
  const [draft, setDraft] = useState("");

  const models =
    providers
      .find((item) => item.provider === chat.provider)
      ?.models.map(({ id }) => id) ?? [];
  const isReady = chat.provider !== "" && chat.model !== "";

  const handleSubmit = () => {
    const content = draft.trim();
    const goals = nonEmptyGoalSets(goalSteps);
    const view: ChatView = {
      space_id: spaceId ?? undefined,
      flow_id: flowId ?? undefined,
      goals: goals.length > 0 ? fromGoalSets(goals) : undefined,
      init: Object.keys(flowInit).length > 0 ? flowInit : undefined,
      locale,
    };
    setDraft("");
    void chat.send(content, view);
  };

  return {
    providers: providers.map((item) => item.provider),
    models,
    provider: chat.provider,
    model: chat.model,
    messages: chat.messages,
    isPending: chat.isPending,
    error: chat.error,
    isComposerDisabled: !isReady,
    draft,
    setDraft,
    setProvider: chat.setProvider,
    setModel: chat.setModel,
    close: chat.toggleOpen,
    handleSubmit,
  };
};
