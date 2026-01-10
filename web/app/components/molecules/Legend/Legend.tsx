import React from "react";
import styles from "./Legend.module.css";
import { useT } from "@/app/i18n";

const Legend: React.FC = () => {
  const t = useT();

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
        <div className={styles.divider}>
          <div className={`${styles.line} ${styles.lineRequired}`}></div>
          <span className={styles.label}>{t("legend.required")}</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.line} ${styles.lineOptional}`}></div>
          <span className={styles.label}>{t("legend.optional")}</span>
        </div>
      </div>
    </div>
  );
};

export default Legend;
