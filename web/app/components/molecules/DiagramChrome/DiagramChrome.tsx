import React from "react";
import {
  Background,
  BackgroundVariant,
  ControlButton,
  Panel,
} from "@xyflow/react";
import {
  IconChat,
  IconFitView,
  IconSignOut,
  IconThemeDark,
  IconThemeLight,
  IconZoomIn,
  IconZoomOut,
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
      <Panel
        position="bottom-right"
        className={glassChromeStyles.controlGroups}
        style={{
          right: "var(--spacing-md)",
          bottom: "var(--spacing-md)",
          margin: 0,
        }}
      >
        <div className={glassChromeStyles.controls}>
          <ControlButton
            onClick={toggleChat}
            title={chatLabel}
            aria-label={chatLabel}
            aria-pressed={isChatOpen}
          >
            <IconChat />
          </ControlButton>
        </div>
        <div className={glassChromeStyles.controls}>
          <ControlButton
            onClick={handleZoomIn}
            title={t("controls.zoomIn")}
            aria-label={t("controls.zoomIn")}
          >
            <IconZoomIn />
          </ControlButton>
          <ControlButton
            onClick={handleZoomOut}
            title={t("controls.zoomOut")}
            aria-label={t("controls.zoomOut")}
          >
            <IconZoomOut />
          </ControlButton>
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
        </div>
        {canSignOut && (
          <div className={glassChromeStyles.controls}>
            <ControlButton
              onClick={() => void signOut()}
              title={signOutLabel}
              aria-label={signOutLabel}
            >
              <IconSignOut />
            </ControlButton>
          </div>
        )}
      </Panel>
    </>
  );
};

export default DiagramChrome;
