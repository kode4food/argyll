import React from "react";
import { AlertTriangle } from "lucide-react";
import Link from "next/link";
import styles from "./NotFoundPage.module.css";

const NotFoundPage: React.FC = () => {
  return (
    <div className={styles.page}>
      <div className={styles.content}>
        <AlertTriangle className={styles.icon} />
        <h1 className={styles.title}>404 - Page Not Found</h1>
        <p className={styles.description}>
          The page you&apos;re looking for doesn&apos;t exist. Check the URL or
          return to the overview.
        </p>
        <Link href="/" className={styles.button}>
          Back to Overview
        </Link>
      </div>
    </div>
  );
};

export default NotFoundPage;
