import React from "react";
import {
  Background,
  BackgroundVariant,
  ControlButton,
  Controls,
} from "@xyflow/react";
import {
  IconChat,
  IconFitView,
  IconSignOut,
  IconThemeDark,
  IconThemeLight,
} from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";
import glassChromeStyles from "@/app/styles/modules/GlassChrome.module.css";
import { useDiagramChrome } from "./useDiagramChrome";

const DiagramChrome: React.FC = () => {
  const t = useT();
  const {
    isDark,
    toggleTheme,
    fitView,
    handleZoomIn,
    handleZoomOut,
    themeLabel,
    isChatOpen,
    toggleChat,
    chatLabel,
    canSignOut,
    signOut,
    signOutLabel,
  } = useDiagramChrome();

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
          right: "var(--spacing-md)",
          bottom: "var(--spacing-md)",
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
          {isDark ? <IconThemeLight /> : <IconThemeDark />}
        </ControlButton>
        <ControlButton
          onClick={toggleChat}
          title={chatLabel}
          aria-label={chatLabel}
          aria-pressed={isChatOpen}
        >
          <IconChat />
        </ControlButton>
        {canSignOut && (
          <ControlButton
            onClick={() => void signOut()}
            title={signOutLabel}
            aria-label={signOutLabel}
          >
            <IconSignOut />
          </ControlButton>
        )}
      </Controls>
    </>
  );
};

export default DiagramChrome;
