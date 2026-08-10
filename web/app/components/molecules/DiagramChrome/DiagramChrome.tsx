import React, { useCallback } from "react";
import {
  Background,
  BackgroundVariant,
  ControlButton,
  Controls,
  useReactFlow,
} from "@xyflow/react";
import {
  IconFitView,
  IconThemeDark,
  IconThemeLight,
} from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";
import { useFitView } from "@/app/hooks/useFitView";
import { useTheme, useToggleTheme } from "@/app/store/themeStore";
import glassChromeStyles from "@/app/styles/modules/GlassChrome.module.css";

const DiagramChrome: React.FC = () => {
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

  return (
    <>
      <Background
        variant={BackgroundVariant.Dots}
        gap={20}
        size={1}
        className="diagram-background"
      />
      <Controls
        className={glassChromeStyles.controls}
        orientation="vertical"
        position="bottom-right"
        showInteractive={false}
        showFitView={false}
        onZoomIn={handleZoomIn}
        onZoomOut={handleZoomOut}
        style={{
          right: "var(--spacing-xl)",
          bottom: "var(--spacing-xl)",
        }}
      >
        <ControlButton
          onClick={fitView}
          title={t("controls.fitView")}
          aria-label={t("controls.fitView")}
        >
          <IconFitView />
        </ControlButton>
        <ControlButton
          onClick={toggleTheme}
          title={themeLabel}
          aria-label={themeLabel}
        >
          {theme === "dark" ? <IconThemeLight /> : <IconThemeDark />}
        </ControlButton>
      </Controls>
    </>
  );
};

export default DiagramChrome;
