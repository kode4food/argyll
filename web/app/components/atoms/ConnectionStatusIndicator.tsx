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
          color: "text-amber-900",
          animate: true,
          semiTransparent: false,
          bgColor: "bg-amber-100",
        };
      case "reconnecting":
        return {
          icon: RefreshCw,
          text: `Reconnecting... (attempt ${reconnectAttempt})`,
          color: "text-amber-900",
          animate: true,
          semiTransparent: false,
          bgColor: "bg-amber-100",
        };
      case "disconnected":
        return {
          icon: WifiOff,
          text: "Disconnected",
          color: "text-gray-500",
          animate: false,
          semiTransparent: false,
          bgColor: "bg-white",
        };
      case "failed":
        return {
          icon: AlertCircle,
          text: "Connection failed",
          color: "text-red-500",
          animate: false,
          semiTransparent: false,
          bgColor: "bg-white",
        };
      default:
        return {
          icon: Wifi,
          text: "Connected",
          color: "text-green-500",
          animate: false,
          semiTransparent: false,
          bgColor: "bg-white",
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;
  const showReconnect =
    (status === "disconnected" || status === "failed") && onReconnect;

  return (
    <div
      className={`${styles.status} ${config.bgColor} ${config.semiTransparent ? styles.statusSemiTransparent : ""}`}
    >
      <Icon
        className={`${styles.icon} ${config.color} ${config.animate ? "animate-spin" : ""}`}
      />
      <span className={`${styles.text} ${config.color}`}>{config.text}</span>
      {showReconnect && (
        <button onClick={onReconnect} className={styles.retryButton}>
          Retry
        </button>
      )}
    </div>
  );
};

export default ConnectionStatusIndicator;
