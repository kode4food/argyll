import React from "react";
import {
  IconError,
  IconAttributeStatusSatisfied,
  IconAttributeOptional,
  IconAttributeStatusSkipped,
  LucideIcon,
} from "@/utils/iconRegistry";
import LazyCodeEditor from "@/app/components/molecules/LazyCodeEditor";
import { useT } from "@/app/i18n";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import { FlowInputStatus, getFlowInputStatus } from "./flowFormUtils";
import styles from "./FlowAttributesSection.module.css";

interface FlowAttributesSectionProps {
  editorMode: "basic" | "json";
  emptyAttributesLabel: string;
  flowInputOptions: FlowInputOption[];
  flowInputValues: Record<string, string>;
  flowInputValuesRaw: Record<string, string>;
  getFlowInputPlaceholder: (option: FlowInputOption) => string;
  handleBasicInputChange: (name: string, value: string) => void;
  initialState: string;
  jsonError: string | null;
  onEditorModeChange: (mode: "basic" | "json") => void;
  onFocusedPreviewAttributeChange: (name: string | null) => void;
  setInitialState: (value: string) => void;
}

const STATUS_ICONS: Record<FlowInputStatus, LucideIcon> = {
  requiredMissing: IconError,
  requiredSatisfied: IconAttributeStatusSatisfied,
  optionalMissing: IconAttributeOptional,
  optionalSatisfied: IconAttributeStatusSatisfied,
  outputSatisfied: IconAttributeStatusSatisfied,
  unreachable: IconAttributeStatusSkipped,
};

const STATUS_CLASSES: Record<FlowInputStatus, string> = {
  requiredMissing: styles.badgeRequired,
  requiredSatisfied: styles.badgeRequired,
  optionalMissing: styles.badgeOptional,
  optionalSatisfied: styles.badgeOptional,
  outputSatisfied: styles.badgeOutput,
  unreachable: styles.badgeMuted,
};

const FlowAttributesSection: React.FC<FlowAttributesSectionProps> = ({
  editorMode,
  emptyAttributesLabel,
  flowInputOptions,
  flowInputValues,
  flowInputValuesRaw,
  getFlowInputPlaceholder,
  handleBasicInputChange,
  initialState,
  jsonError,
  onEditorModeChange,
  onFocusedPreviewAttributeChange,
  setInitialState,
}) => {
  const t = useT();

  const statusLabels: Record<FlowInputStatus, string> = {
    requiredMissing: t("flowCreate.badgeRequiredMissing"),
    requiredSatisfied: t("flowCreate.badgeRequiredSatisfied"),
    optionalMissing: t("flowCreate.badgeOptionalMissing"),
    optionalSatisfied: t("flowCreate.badgeOptionalSatisfied"),
    outputSatisfied: t("flowCreate.badgeOutputSatisfied"),
    unreachable: t("flowCreate.badgeUnreachable"),
  };

  return (
    <section className={`${styles.sectionCard} ${styles.attributesSection}`}>
      <div className={styles.sectionHeader}>
        <div className={styles.sectionTitle}>
          {t("flowCreate.requiredAttributesLabel")}
        </div>
        <div className={styles.editorModeToggleGroup}>
          <button
            type="button"
            className={`${styles.editorModeToggle} ${
              editorMode === "basic" ? styles.editorModeToggleActive : ""
            }`}
            onClick={() => onEditorModeChange("basic")}
          >
            {t("flowCreate.modeBasic")}
          </button>
          <button
            type="button"
            className={`${styles.editorModeToggle} ${
              editorMode === "json" ? styles.editorModeToggleActive : ""
            }`}
            onClick={() => onEditorModeChange("json")}
          >
            {t("flowCreate.modeJson")}
          </button>
        </div>
      </div>
      <div className={styles.editorContainer}>
        {editorMode === "basic" ? (
          <div className={styles.editorWrapper}>
            {flowInputOptions.length === 0 ? (
              <div className={styles.emptyAttributesCentered}>
                {emptyAttributesLabel}
              </div>
            ) : (
              <div className={styles.attributeTableScroll}>
                <div className={styles.attributeList}>
                  {flowInputOptions.map((option) => {
                    const value = flowInputValues[option.name] || "";
                    const rawValue = flowInputValuesRaw[option.name] || "";
                    const status = getFlowInputStatus(option, rawValue);
                    const Icon = STATUS_ICONS[status];
                    const label = statusLabels[status];

                    return (
                      <div
                        key={option.name}
                        className={styles.attributeListItem}
                      >
                        <div className={styles.badgeCell}>
                          <span
                            className={`${styles.statusBadge} ${STATUS_CLASSES[status]}`}
                            role="img"
                            aria-label={label}
                            title={label}
                          >
                            <Icon size={16} aria-hidden />
                          </span>
                        </div>
                        <div className={styles.attributeNameCell}>
                          <span className={styles.attributeNameText}>
                            {option.name}
                          </span>
                        </div>
                        <div className={styles.attributeValueCell}>
                          <input
                            type="text"
                            value={value}
                            onChange={(e) =>
                              handleBasicInputChange(
                                option.name,
                                e.target.value
                              )
                            }
                            onFocus={() =>
                              onFocusedPreviewAttributeChange(option.name)
                            }
                            className={styles.attributeValueInput}
                            placeholder={getFlowInputPlaceholder(option)}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        ) : (
          <>
            <div className={styles.editorWrapper}>
              <LazyCodeEditor
                value={initialState}
                onChange={setInitialState}
                height="100%"
              />
            </div>
            {jsonError && (
              <div className={styles.jsonError}>
                {t("flowCreate.invalidJson", { error: jsonError })}
              </div>
            )}
          </>
        )}
      </div>
    </section>
  );
};

export default FlowAttributesSection;
