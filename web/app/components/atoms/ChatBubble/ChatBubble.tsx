import React from "react";
import styles from "./ChatBubble.module.css";

export type ChatBubbleVariant = "user" | "assistant" | "error";

export interface ChatBubbleProps {
  variant: ChatBubbleVariant;
  children: React.ReactNode;
}

const ChatBubble: React.FC<ChatBubbleProps> = ({ variant, children }) => (
  <div className={`${styles.bubble} ${styles[variant]}`}>{children}</div>
);

export default ChatBubble;
