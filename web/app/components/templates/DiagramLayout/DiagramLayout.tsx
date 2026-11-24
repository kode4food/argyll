import React from "react";
import ChatBox from "@/app/components/organisms/ChatBox";
import { useChatOpen } from "@/app/store/chatStore";
import styles from "./DiagramLayout.module.css";

interface DiagramLayoutProps {
  children: React.ReactNode;
  className?: string;
  containerRef?: React.Ref<HTMLDivElement>;
}

const DiagramLayout: React.FC<DiagramLayoutProps> = ({
  children,
  className,
  containerRef,
}) => {
  const isChatOpen = useChatOpen();
  const containerClassName = [styles.container, className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={containerClassName}>
      <div className={styles.diagramContainer} ref={containerRef}>
        {children}
        {isChatOpen && <ChatBox />}
      </div>
    </div>
  );
};

export default DiagramLayout;
