import React, { useCallback, useEffect, useState } from "react";
import Spinner from "@/app/components/atoms/Spinner";
import ErrorFallback from "@/app/components/molecules/ErrorFallback";
import { useT } from "@/app/i18n";
import { useAuthStore } from "@/app/store/authStore";
import styles from "./AuthGate.module.css";

interface AuthGateProps {
  children: React.ReactNode;
}

// Renders the app once the user is signed in, or right away when login is off
const AuthGate: React.FC<AuthGateProps> = ({ children }) => {
  const t = useT();
  const isReady = useAuthStore((state) => state.isReady);
  const start = useAuthStore((state) => state.start);
  const login = useAuthStore((state) => state.login);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    start().catch((err: unknown) =>
      setError(err instanceof Error ? err : new Error(String(err)))
    );
  }, [start]);

  const handleRetry = useCallback(() => {
    setError(null);
    void login().catch((err: unknown) =>
      setError(err instanceof Error ? err : new Error(String(err)))
    );
  }, [login]);

  if (error) {
    return (
      <ErrorFallback
        error={error}
        resetError={handleRetry}
        title={t("auth.failedTitle")}
        description={t("auth.failedDescription")}
      />
    );
  }
  if (!isReady) {
    return (
      <div
        className={styles.pending}
        role="status"
        aria-label={t("auth.signingIn")}
      >
        <Spinner />
      </div>
    );
  }
  return <>{children}</>;
};

export default AuthGate;
