import React from "react";
import { IconChatSend } from "@/utils/iconRegistry";
import styles from "./ChatComposer.module.css";

export interface ChatComposerProps {
  value: string;
  isDisabled: boolean;
  placeholder: string;
  sendLabel: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
}

const COMPOSER_ROWS = 2;

const ChatComposer: React.FC<ChatComposerProps> = ({
  value,
  isDisabled,
  placeholder,
  sendLabel,
  onChange,
  onSubmit,
}) => {
  const canSend = !isDisabled && value.trim() !== "";

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (canSend) onSubmit();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== "Enter" || e.shiftKey) return;
    e.preventDefault();
    if (canSend) onSubmit();
  };

  return (
    <form className={styles.composer} onSubmit={handleSubmit}>
      <textarea
        className={styles.input}
        value={value}
        rows={COMPOSER_ROWS}
        disabled={isDisabled}
        placeholder={placeholder}
        aria-label={placeholder}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      <button
        type="submit"
        className={styles.send}
        disabled={!canSend}
        title={sendLabel}
        aria-label={sendLabel}
      >
        <IconChatSend />
      </button>
    </form>
  );
};

export default ChatComposer;
