import React from "react";
import { IconCompensate, IconRetry, IconStartFlow } from "@/utils/iconRegistry";
import Spinner from "@/app/components/atoms/Spinner";
import IconCheckbox from "@/app/components/molecules/IconCheckbox";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import { useT } from "@/app/i18n";
import styles from "./FlowStartSection.module.css";

interface FlowStartSectionProps {
  compensate: boolean;
  creating: boolean;
  disabled: boolean;
  flowId: string;
  onCompensateChange: (value: boolean) => void;
  onCreateFlow: () => void | Promise<void>;
  onFlowIdChange: (value: string) => void;
  onGenerateId: () => void;
}

const FlowStartSection: React.FC<FlowStartSectionProps> = ({
  compensate,
  creating,
  disabled,
  flowId,
  onCompensateChange,
  onCreateFlow,
  onFlowIdChange,
  onGenerateId,
}) => {
  const t = useT();

  return (
    <fieldset className={styles.section} disabled={disabled}>
      <div className={styles.labelRow}>
        <label className={styles.label}>
          <span className={styles.labelIcon}>
            <IconStartFlow aria-hidden="true" />
          </span>
          {t("flowCreate.startFlowLabel")}
        </label>
        <IconCheckbox
          checked={compensate}
          Icon={IconCompensate}
          label={t("flowCreate.compensateLabel")}
          onChange={onCompensateChange}
          title={t("flowCreate.compensateTitle")}
        />
      </div>
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
          disabled={!flowId.trim()}
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
    </fieldset>
  );
};

export default FlowStartSection;
