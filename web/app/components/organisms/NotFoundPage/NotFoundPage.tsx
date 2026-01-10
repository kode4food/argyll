import React from "react";
import { AlertTriangle } from "lucide-react";
import { Link } from "react-router-dom";
import styles from "./NotFoundPage.module.css";
import { useT } from "@/app/i18n";

const NotFoundPage: React.FC = () => {
  const t = useT();

  return (
    <div className={styles.page}>
      <div className={styles.content}>
        <AlertTriangle className={styles.icon} />
        <h1 className={styles.title}>{t("notFound.title")}</h1>
        <p className={styles.description}>{t("notFound.description")}</p>
        <Link to="/" className={styles.button}>
          {t("common.backToOverview")}
        </Link>
      </div>
    </div>
  );
};

export default NotFoundPage;
