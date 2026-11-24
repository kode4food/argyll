import React, { useEffect, useRef } from "react";
import type { ChatMessage } from "@/app/api";
import ChatBubble from "@/app/components/atoms/ChatBubble";
import Spinner from "@/app/components/atoms/Spinner";
import MarkdownText from "@/app/components/molecules/MarkdownText";
import styles from "./ChatTranscript.module.css";

export interface ChatTranscriptProps {
  messages: ChatMessage[];
  isPending: boolean;
  error: string | null;
  emptyText: string;
}

const ChatTranscript: React.FC<ChatTranscriptProps> = ({
  messages,
  isPending,
  error,
  emptyText,
}) => {
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages.length, isPending, error]);

  return (
    <div ref={listRef} className={styles.transcript} aria-live="polite">
      {messages.length === 0 && !isPending && (
        <p className={styles.empty}>{emptyText}</p>
      )}
      {messages.map((message, idx) => (
        <ChatBubble key={idx} variant={message.role}>
          {message.role === "assistant" ? (
            <MarkdownText source={message.content} />
          ) : (
            message.content
          )}
        </ChatBubble>
      ))}
      {isPending && (
        <div className={styles.pending}>
          <Spinner size="sm" />
        </div>
      )}
      {error && (
        <div role="alert" className={styles.errorRow}>
          <ChatBubble variant="error">{error}</ChatBubble>
        </div>
      )}
    </div>
  );
};

export default ChatTranscript;
