import React from "react";
import LazyCodeEditor from "@/app/components/molecules/LazyCodeEditor";
import { useT } from "@/app/i18n";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import { FlowInputStatus } from "./flowFormUtils";
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
  statusLabelByType: Record<FlowInputStatus, string>;
  toFlowInputStatus: (
    option: FlowInputOption,
    value: string
  ) => FlowInputStatus;
}

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
  statusLabelByType,
  toFlowInputStatus,
}) => {
  const t = useT();
  const statusClassByType: Record<FlowInputStatus, string> = {
    provided: styles.requiredBadgeSatisfied,
    defaulted: styles.requiredBadgeDefault,
    required: styles.requiredBadgeMissing,
    optional: styles.requiredBadgeOptional,
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
                    const status = toFlowInputStatus(option, rawValue);
                    const statusClass = statusClassByType[status];
                    const statusLabel = statusLabelByType[status];

                    return (
                      <div
                        key={option.name}
                        className={styles.attributeListItem}
                      >
                        <div className={styles.requiredBadgeCell}>
                          <span
                            className={`${styles.requiredBadge} ${statusClass}`}
                            role="img"
                            aria-label={statusLabel}
                            title={statusLabel}
                          />
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
