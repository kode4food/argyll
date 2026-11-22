import React from "react";
import { IconAttributes } from "@/utils/iconRegistry";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import EditorModeToggle from "@/app/components/atoms/EditorModeToggle";
import useArrowFocus from "@/app/hooks/useArrowFocus";
import { useT } from "@/app/i18n";
import { useTheme } from "@/app/store/themeStore";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import styles from "./FlowAttributesSection.module.css";
import { useFlowInputRows } from "./useFlowInputRows";

interface FlowAttributesSectionProps {
  editorMode: "basic" | "json";
  emptyAttributesLabel: string;
  flowInputOptions: FlowInputOption[];
  flowInputValues: Record<string, string>;
  flowInputValuesRaw: Record<string, string>;
  getFlowInputPlaceholder: (option: FlowInputOption) => string;
  handleBasicInputBlur: () => void;
  handleBasicInputChange: (name: string, value: string) => void;
  initialState: string;
  jsonError: string | null;
  onEditorModeChange: (mode: "basic" | "json") => void;
  onFocusedPreviewAttributeChange: (name: string | null) => void;
  setInitialState: (value: string) => void;
}

const FlowAttributesSection: React.FC<FlowAttributesSectionProps> = ({
  editorMode,
  emptyAttributesLabel,
  flowInputOptions,
  flowInputValues,
  flowInputValuesRaw,
  getFlowInputPlaceholder,
  handleBasicInputBlur,
  handleBasicInputChange,
  initialState,
  jsonError,
  onEditorModeChange,
  onFocusedPreviewAttributeChange,
  setInitialState,
}) => {
  const t = useT();
  const theme = useTheme();
  const handleArrowFocus = useArrowFocus();

  const rows = useFlowInputRows(
    flowInputOptions,
    flowInputValues,
    flowInputValuesRaw
  );

  return (
    <section className={`${styles.sectionCard} ${styles.attributesSection}`}>
      <div className={styles.sectionHeader}>
        <div className={styles.sectionTitle}>
          <span className={styles.sectionTitleIcon}>
            <IconAttributes aria-hidden="true" />
          </span>
          {t("flowCreate.requiredAttributesLabel")}
        </div>
        <EditorModeToggle
          editorMode={editorMode}
          onChange={onEditorModeChange}
          basicLabel={t("flowCreate.modeBasic")}
          jsonLabel={t("flowCreate.modeJson")}
        />
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
                <div
                  className={styles.attributeList}
                  onKeyDown={handleArrowFocus}
                >
                  {rows.map(({ option, value, Icon, className, label }) => {
                    return (
                      <div
                        key={option.name}
                        className={styles.attributeListItem}
                      >
                        <div className={styles.badgeCell}>
                          <span
                            className={`${styles.statusBadge} ${className}`}
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
                            data-arrow-focus-item="true"
                            value={value}
                            onChange={(e) =>
                              handleBasicInputChange(
                                option.name,
                                e.target.value
                              )
                            }
                            onBlur={handleBasicInputBlur}
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
              <ScriptEditor
                value={initialState}
                onChange={setInitialState}
                height="100%"
                language="json"
                theme={theme}
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
