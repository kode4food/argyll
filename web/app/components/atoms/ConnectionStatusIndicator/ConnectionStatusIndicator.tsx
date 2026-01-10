import React from "react";
import { Wifi, WifiOff, RefreshCw, AlertCircle } from "lucide-react";
import { ConnectionStatus } from "@/app/types/websocket";
import styles from "./ConnectionStatusIndicator.module.css";
import { useT } from "@/app/i18n";

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  reconnectAttempt?: number;
  onReconnect?: () => void;
}

const ConnectionStatusIndicator: React.FC<ConnectionStatusIndicatorProps> = ({
  status,
  reconnectAttempt = 0,
  onReconnect,
}) => {
  const t = useT();

  if (status === "connected") {
    return null;
  }

  const getStatusConfig = () => {
    switch (status) {
      case "connecting":
        return {
          icon: RefreshCw,
          text: t("connectionStatus.connecting"),
          colorClass: styles.colorWarning,
          animate: true,
          semiTransparent: false,
          bgClass: styles.bgWarning,
        };
      case "reconnecting":
        return {
          icon: RefreshCw,
          text: t("connectionStatus.reconnecting", {
            attempt: reconnectAttempt,
          }),
          colorClass: styles.colorWarning,
          animate: true,
          semiTransparent: false,
          bgClass: styles.bgWarning,
        };
      case "disconnected":
        return {
          icon: WifiOff,
          text: t("connectionStatus.disconnected"),
          colorClass: styles.colorNeutral,
          animate: false,
          semiTransparent: false,
          bgClass: styles.bgWhite,
        };
      case "failed":
        return {
          icon: AlertCircle,
          text: t("connectionStatus.failed"),
          colorClass: styles.colorError,
          animate: false,
          semiTransparent: false,
          bgClass: styles.bgWhite,
        };
      default:
        return {
          icon: Wifi,
          text: t("connectionStatus.connected"),
          colorClass: styles.colorSuccess,
          animate: false,
          semiTransparent: false,
          bgClass: styles.bgWhite,
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;
  const showReconnect =
    (status === "disconnected" || status === "failed") && onReconnect;

  return (
    <div
      className={`${styles.status} ${config.bgClass} ${config.semiTransparent ? styles.statusSemiTransparent : ""}`}
    >
      <Icon
        className={`${styles.icon} ${config.colorClass} ${config.animate ? "animate-spin" : ""}`}
      />
      <span className={`${styles.text} ${config.colorClass}`}>
        {config.text}
      </span>
      {showReconnect && (
        <button onClick={onReconnect} className={styles.retryButton}>
          {t("common.retry")}
        </button>
      )}
    </div>
  );
};

export default ConnectionStatusIndicator;
