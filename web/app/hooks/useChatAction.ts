import { useEffect } from "react";
import type { ChatAction, ChatActionType } from "@/app/api";
import { useChatStore } from "@/app/store/chatStore";

type ChatActionHandlers = {
  [K in ChatActionType]?: (action: Extract<ChatAction, { type: K }>) => void;
};

export const useChatAction = (handlers: ChatActionHandlers) => {
  const action = useChatStore((state) => state.action);
  const clearAction = useChatStore((state) => state.clearAction);

  useEffect(() => {
    if (!action) return;
    const handle = handlers[action.type] as
      ((action: ChatAction) => void) | undefined;
    if (!handle) return;
    clearAction();
    handle(action);
  }, [action, clearAction, handlers]);
};
