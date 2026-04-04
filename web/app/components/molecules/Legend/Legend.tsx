import React from "react";
import {
  IconFitView,
  IconThemeDark,
  IconThemeLight,
  IconZoomIn,
  IconZoomOut,
} from "@/utils/iconRegistry";
import type { Theme } from "@/app/store/themeStore";
import styles from "./Legend.module.css";
import { useT } from "@/app/i18n";

export interface LegendProps {
  onZoomIn?: () => void;
  onZoomOut?: () => void;
  onFitView?: () => void;
  onToggleTheme?: () => void;
  theme?: Theme;
}

const Legend: React.FC<LegendProps> = ({
  onZoomIn,
  onZoomOut,
  onFitView,
  onToggleTheme,
  theme = "light",
}) => {
  const t = useT();
  const showActions =
    !!onZoomIn || !!onZoomOut || !!onFitView || !!onToggleTheme;
  const ZoomInIcon = IconZoomIn;
  const ZoomOutIcon = IconZoomOut;
  const FitViewIcon = IconFitView;

  return (
    <div className={styles.root}>
      <div className={styles.content}>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxResolver}`}></div>
          <span className={styles.label}>{t("legend.resolver")}</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxProcessor}`}></div>
          <span className={styles.label}>{t("legend.processor")}</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxCollector}`}></div>
          <span className={styles.label}>{t("legend.collector")}</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxStandalone}`}></div>
          <span className={styles.label}>{t("legend.standalone")}</span>
        </div>
        <div className={styles.divider}>
          <div className={`${styles.line} ${styles.lineRequired}`}></div>
          <span className={styles.label}>{t("legend.required")}</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.line} ${styles.lineOptional}`}></div>
          <span className={styles.label}>{t("legend.optional")}</span>
        </div>
      </div>
      {showActions && (
        <div className={styles.actionsSection}>
          <div className={styles.actions}>
            {onZoomOut && (
              <button
                type="button"
                className={styles.actionButton}
                onClick={onZoomOut}
                title={t("legend.zoomOut")}
                aria-label={t("legend.zoomOut")}
              >
                <ZoomOutIcon className={styles.actionIcon} />
              </button>
            )}
            {onZoomIn && (
              <button
                type="button"
                className={styles.actionButton}
                onClick={onZoomIn}
                title={t("legend.zoomIn")}
                aria-label={t("legend.zoomIn")}
              >
                <ZoomInIcon className={styles.actionIcon} />
              </button>
            )}
            {onFitView && (
              <button
                type="button"
                className={styles.actionButton}
                onClick={onFitView}
                title={t("legend.autoZoom")}
                aria-label={t("legend.autoZoom")}
              >
                <FitViewIcon className={styles.actionIcon} />
              </button>
            )}
            {onToggleTheme && (
              <button
                type="button"
                className={styles.actionButton}
                onClick={onToggleTheme}
                title={
                  theme === "dark"
                    ? t("legend.switchToLightMode")
                    : t("legend.switchToDarkMode")
                }
                aria-label={
                  theme === "dark"
                    ? t("legend.switchToLightMode")
                    : t("legend.switchToDarkMode")
                }
              >
                {theme === "dark" ? (
                  <IconThemeLight className={styles.actionIcon} />
                ) : (
                  <IconThemeDark className={styles.actionIcon} />
                )}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default Legend;
