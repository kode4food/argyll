import { useCallback } from "react";
import { useReactFlow } from "@xyflow/react";
import { useT } from "@/app/i18n";
import { useFitView } from "@/app/hooks/useFitView";
import { useTheme, useToggleTheme } from "@/app/store/themeStore";
import { useChatOpen, useToggleChat } from "@/app/store/chatStore";
import { useAuthStore } from "@/app/store/authStore";

export const useDiagramChrome = () => {
  const t = useT();
  const theme = useTheme();
  const toggleTheme = useToggleTheme();
  const isChatOpen = useChatOpen();
  const toggleChat = useToggleChat();
  const canSignOut = useAuthStore((state) => state.isEnabled);
  const userName = useAuthStore((state) => state.userName);
  const signOut = useAuthStore((state) => state.logout);
  const fitView = useFitView();
  const reactFlowInstance = useReactFlow();

  const handleZoomIn = useCallback(() => {
    void reactFlowInstance.zoomIn();
  }, [reactFlowInstance]);
  const handleZoomOut = useCallback(() => {
    void reactFlowInstance.zoomOut();
  }, [reactFlowInstance]);

  const themeLabel =
    theme === "dark"
      ? t("controls.switchToLightMode")
      : t("controls.switchToDarkMode");

  return {
    isDark: theme === "dark",
    toggleTheme,
    fitView,
    handleZoomIn,
    handleZoomOut,
    themeLabel,
    isChatOpen,
    toggleChat,
    chatLabel: isChatOpen ? t("chat.close") : t("chat.open"),
    canSignOut,
    signOut,
    signOutLabel: t("auth.signOut", { user: userName ?? "" }),
  };
};
