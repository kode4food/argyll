import { useCallback } from "react";
import { useReactFlow } from "@xyflow/react";
import { useT } from "@/app/i18n";
import { useFitView } from "@/app/hooks/useFitView";
import { useTheme, useToggleTheme } from "@/app/store/themeStore";

export const useDiagramChrome = () => {
  const t = useT();
  const theme = useTheme();
  const toggleTheme = useToggleTheme();
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
  };
};
