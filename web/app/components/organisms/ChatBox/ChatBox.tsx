import React from "react";
import ChatComposer from "@/app/components/molecules/ChatComposer";
import ChatModelPicker from "@/app/components/molecules/ChatModelPicker";
import ChatTranscript from "@/app/components/molecules/ChatTranscript";
import { useT } from "@/app/i18n";
import { IconChat, IconClose } from "@/utils/iconRegistry";
import styles from "./ChatBox.module.css";
import { useChatBox } from "./useChatBox";

const ChatBox: React.FC = () => {
  const t = useT();
  const chat = useChatBox();

  return (
    <section className={styles.box} aria-label={t("chat.title")}>
      <header className={styles.header}>
        <span className={styles.title}>
          <IconChat aria-hidden="true" />
          {t("chat.title")}
        </span>
        <button
          type="button"
          className={styles.close}
          onClick={chat.close}
          title={t("chat.close")}
          aria-label={t("chat.close")}
        >
          <IconClose />
        </button>
      </header>
      <ChatModelPicker
        providers={chat.providers}
        models={chat.models}
        provider={chat.provider}
        model={chat.model}
        providerLabel={t("chat.provider")}
        modelLabel={t("chat.model")}
        onProviderChange={chat.setProvider}
        onModelChange={chat.setModel}
      />
      <ChatTranscript
        messages={chat.messages}
        isPending={chat.isPending}
        error={chat.error}
        emptyText={t("chat.empty")}
      />
      <ChatComposer
        value={chat.draft}
        isDisabled={chat.isComposerDisabled}
        placeholder={t("chat.placeholder")}
        sendLabel={t("chat.send")}
        onChange={chat.setDraft}
        onSubmit={chat.handleSubmit}
      />
    </section>
  );
};

export default ChatBox;
