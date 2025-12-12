import React from "react";
import { Wifi, WifiOff, RefreshCw, AlertCircle } from "lucide-react";
import { ConnectionStatus } from "../../hooks/useWebSocketContext";
import styles from "./ConnectionStatusIndicator.module.css";

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
  if (status === "connected") {
    return null;
  }

  const getStatusConfig = () => {
    switch (status) {
      case "connecting":
        return {
          icon: RefreshCw,
          text: "Connecting...",
          colorClass: styles.colorWarning,
          animate: true,
          semiTransparent: false,
          bgClass: styles.bgWarning,
        };
      case "reconnecting":
        return {
          icon: RefreshCw,
          text: `Reconnecting... (attempt ${reconnectAttempt})`,
          colorClass: styles.colorWarning,
          animate: true,
          semiTransparent: false,
          bgClass: styles.bgWarning,
        };
      case "disconnected":
        return {
          icon: WifiOff,
          text: "Disconnected",
          colorClass: styles.colorNeutral,
          animate: false,
          semiTransparent: false,
          bgClass: styles.bgWhite,
        };
      case "failed":
        return {
          icon: AlertCircle,
          text: "Connection failed",
          colorClass: styles.colorError,
          animate: false,
          semiTransparent: false,
          bgClass: styles.bgWhite,
        };
      default:
        return {
          icon: Wifi,
          text: "Connected",
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
          Retry
        </button>
      )}
    </div>
  );
};

export default ConnectionStatusIndicator;
