import React from "react";
import { AlertCircle, RefreshCw } from "lucide-react";
import styles from "./ErrorFallback.module.css";

interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
  title?: string;
  description?: string;
}

const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  resetError,
  title = "Something went wrong",
  description = "An unexpected error occurred. You can try reloading this section.",
}) => {
  return (
    <div className={styles.fallback}>
      <div className={styles.content}>
        <AlertCircle className={styles.icon} />
        <h2 className={styles.title}>{title}</h2>
        <p className={styles.description}>{description}</p>
        <details className={styles.details}>
          <summary className={styles.detailsSummary}>Error details</summary>
          <pre className={styles.detailsPre}>
            {error.message}
            {error.stack && `\n\n${error.stack}`}
          </pre>
        </details>
        <button onClick={resetError} className={styles.button}>
          <RefreshCw className={styles.buttonIcon} />
          Try again
        </button>
      </div>
    </div>
  );
};

export default ErrorFallback;
