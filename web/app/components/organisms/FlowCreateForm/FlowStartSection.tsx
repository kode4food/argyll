import React from "react";
import { IconRetry, IconStartFlow } from "@/utils/iconRegistry";
import Spinner from "@/app/components/atoms/Spinner";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import { useT } from "@/app/i18n";
import styles from "./FlowStartSection.module.css";

interface FlowStartSectionProps {
  creating: boolean;
  disableStart: boolean;
  flowId: string;
  onCreateFlow: () => void | Promise<void>;
  onFlowIdChange: (value: string) => void;
  onGenerateId: () => void;
}

const FlowStartSection: React.FC<FlowStartSectionProps> = ({
  creating,
  disableStart,
  flowId,
  onCreateFlow,
  onFlowIdChange,
  onGenerateId,
}) => {
  const t = useT();

  return (
    <div className={styles.section}>
      <label className={styles.label}>{t("flowCreate.startFlowLabel")}</label>
      <div className={styles.footerRow}>
        <SegmentedGroup className={styles.idGroup}>
          <input
            type="text"
            value={flowId}
            onChange={(e) => onFlowIdChange(e.target.value)}
            placeholder={t("flowCreate.flowIdPlaceholder")}
            className={styles.idInputInline}
          />
          <button
            type="button"
            onClick={onGenerateId}
            className={styles.buttonGenerateSegment}
            title={t("flowCreate.generateIdTitle")}
            aria-label={t("flowCreate.generateIdAria")}
          >
            <IconRetry className={styles.startIcon} />
          </button>
        </SegmentedGroup>
        <button
          onClick={onCreateFlow}
          disabled={disableStart}
          className={`${styles.buttonStart} ${styles.footerIconButton}`}
          title={t("common.start")}
          aria-label={t("common.start")}
        >
          {creating ? (
            <Spinner size="sm" color="white" />
          ) : (
            <IconStartFlow className={styles.startIcon} />
          )}
        </button>
      </div>
    </div>
  );
};

export default FlowStartSection;
