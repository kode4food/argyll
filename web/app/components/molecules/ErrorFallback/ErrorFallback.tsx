import React from "react";
import { IconError, IconRetry } from "@/utils/iconRegistry";
import styles from "./ErrorFallback.module.css";
import { useT } from "@/app/i18n";

interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
  title?: string;
  description?: string;
}

const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  resetError,
  title,
  description,
}) => {
  const t = useT();
  const fallbackTitle = title ?? t("errorFallback.title");
  const fallbackDescription = description ?? t("errorFallback.description");

  return (
    <div className={styles.fallback}>
      <div className={styles.content}>
        <IconError className={styles.icon} />
        <h2 className={styles.title}>{fallbackTitle}</h2>
        <p className={styles.description}>{fallbackDescription}</p>
        <details className={styles.details}>
          <summary className={styles.detailsSummary}>
            {t("errorFallback.details")}
          </summary>
          <pre className={styles.detailsPre}>
            {error.message}
            {error.stack && `\n\n${error.stack}`}
          </pre>
        </details>
        <button onClick={resetError} className={styles.button}>
          <IconRetry className={styles.buttonIcon} />
          {t("common.tryAgain")}
        </button>
      </div>
    </div>
  );
};

export default ErrorFallback;
