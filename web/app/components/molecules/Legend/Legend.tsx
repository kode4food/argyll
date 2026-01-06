import React from "react";
import styles from "./Legend.module.css";

const Legend: React.FC = () => {
  return (
    <div className={styles.root}>
      <div className={styles.content}>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxResolver}`}></div>
          <span className={styles.label}>Resolver Steps</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxProcessor}`}></div>
          <span className={styles.label}>Processor Steps</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.box} ${styles.boxCollector}`}></div>
          <span className={styles.label}>Collector Steps</span>
        </div>
        <div className={styles.divider}>
          <div className={`${styles.line} ${styles.lineRequired}`}></div>
          <span className={styles.label}>Required</span>
        </div>
        <div className={styles.item}>
          <div className={`${styles.line} ${styles.lineOptional}`}></div>
          <span className={styles.label}>Optional</span>
        </div>
      </div>
    </div>
  );
};

export default Legend;
