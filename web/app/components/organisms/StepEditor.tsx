"use client";

import React, { useEffect, createContext, useContext } from "react";
import { createPortal } from "react-dom";
import {
  FileCode2,
  Globe,
  Webhook,
  Trash2,
  Plus,
  Layers,
  Square,
} from "lucide-react";
import { Step, AttributeType, StepType } from "@/app/api";
import ScriptConfigEditor from "../molecules/ScriptConfigEditor";
import DurationInput from "../molecules/DurationInput";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { useStepEditorForm } from "./StepEditor/useStepEditorForm";
import { useModalDimensions } from "./StepEditor/useModalDimensions";
import { Attribute, getAttributeIconProps } from "./StepEditor/stepEditorUtils";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement | null>;
}

interface StepEditingContextValue {
  stepId: string;
  name: string;
  stepType: StepType;
  isCreateMode: boolean;
  setStepId: (value: string) => void;
  setName: (value: string) => void;
  setStepType: (value: StepType) => void;
  attributes: Attribute[];
  addAttribute: () => void;
  updateAttribute: (id: string, field: keyof Attribute, value: any) => void;
  removeAttribute: (id: string) => void;
  cycleAttributeType: (
    id: string,
    currentType: "input" | "optional" | "output"
  ) => void;
  endpoint: string;
  setEndpoint: (value: string) => void;
  healthCheck: string;
  setHealthCheck: (value: string) => void;
  httpTimeout: number;
  setHttpTimeout: (value: number) => void;
}

const ATTRIBUTE_TYPES: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

const StepEditingContext = createContext<StepEditingContextValue | null>(null);

const useStepEditingContext = (): StepEditingContextValue => {
  const ctx = useContext(StepEditingContext);
  if (!ctx) {
    throw new Error(
      "useStepEditingContext must be used within a StepEditor provider"
    );
  }
  return ctx;
};

const BasicFields: React.FC = () => {
  const {
    stepId,
    name,
    stepType,
    isCreateMode,
    setStepId,
    setName,
    setStepType,
  } = useStepEditingContext();

  return (
    <div className={formStyles.row}>
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.label}>Step ID</label>
        <input
          type="text"
          value={stepId}
          onChange={(e) => setStepId(e.target.value)}
          className={formStyles.formControl}
          disabled={!isCreateMode}
          placeholder="my-step"
        />
      </div>
      <div className={`${formStyles.field} ${formStyles.flex2}`}>
        <label className={formStyles.label}>Step Name</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className={formStyles.formControl}
          placeholder="My Step"
        />
      </div>
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.label}>Type</label>
        <div className={formStyles.typeButtonGroup}>
          {[
            {
              type: "sync" as StepType,
              Icon: Globe,
              label: "Sync",
              title: "Synchronous HTTP",
            },
            {
              type: "async" as StepType,
              Icon: Webhook,
              label: "Async",
              title: "Asynchronous HTTP",
            },
            {
              type: "script" as StepType,
              Icon: FileCode2,
              label: "Script",
              title: "Script (Ale)",
            },
          ].map(({ type, Icon, label, title }) => (
            <button
              key={type}
              type="button"
              onClick={(e) => {
                setStepType(type);
                e.currentTarget.blur();
              }}
              className={`${formStyles.typeButton} ${stepType === type ? formStyles.typeButtonActive : ""}`}
              title={title}
            >
              <Icon className={styles.iconSm} />
              <span>{label}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};

const AttributesSection: React.FC = () => {
  const {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    cycleAttributeType,
  } = useStepEditingContext();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>Attributes</label>
        <button
          onClick={addAttribute}
          className={`${formStyles.iconButton} ${formStyles.addButtonStyle}`}
          title="Add attribute"
        >
          <Plus className={styles.iconMd} />
        </button>
      </div>
      <div className={formStyles.argList}>
        {attributes.length === 0 && (
          <div
            className={`${formStyles.attrRow} ${formStyles.attrPlaceholder}`}
            aria-hidden
          >
            <div className={formStyles.attrRowInputs}>
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderIcon}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSelect}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderInput}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderSmall}`}
              />
              <div
                className={`${formStyles.placeholderControl} ${formStyles.placeholderButton}`}
              />
            </div>
            <div className={formStyles.placeholderHint}>
              Attributes describe how steps share data with each other
            </div>
          </div>
        )}
        {attributes.map((attr) => (
          <div key={attr.id} className={formStyles.attrRow}>
            <div className={formStyles.attrRowInputs}>
              <button
                type="button"
                onClick={() => cycleAttributeType(attr.id, attr.attrType)}
                className={`${formStyles.iconButton} ${formStyles.attrIconButtonStyle}`}
                title={`Click to cycle type (current: ${attr.attrType})`}
              >
                {(() => {
                  const { Icon, className } = getAttributeIconProps(
                    attr.attrType
                  );
                  return <Icon className={`${styles.iconMd} ${className}`} />;
                })()}
              </button>
              <select
                value={attr.dataType}
                onChange={(e) =>
                  updateAttribute(attr.id, "dataType", e.target.value)
                }
                className={formStyles.argType}
              >
                {ATTRIBUTE_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
              <input
                type="text"
                value={attr.name}
                onChange={(e) =>
                  updateAttribute(attr.id, "name", e.target.value)
                }
                placeholder="name"
                className={formStyles.argInput}
              />
              {attr.attrType === "optional" && (
                <input
                  type="text"
                  value={attr.defaultValue || ""}
                  onChange={(e) =>
                    updateAttribute(attr.id, "defaultValue", e.target.value)
                  }
                  placeholder="default value"
                  className={formStyles.argInput}
                  title="Default value for optional argument"
                />
              )}
              {attr.attrType !== "output" &&
                attr.dataType === AttributeType.Array && (
                  <div className={formStyles.forEachToggleGroup}>
                    <button
                      type="button"
                      onClick={(e) => {
                        updateAttribute(attr.id, "forEach", false);
                        e.currentTarget.blur();
                      }}
                      className={`${formStyles.forEachToggle} ${!attr.forEach ? formStyles.forEachToggleActive : ""}`}
                      title="Process array as single value"
                    >
                      <Square className={styles.iconSm} />
                      <span>Single</span>
                    </button>
                    <button
                      type="button"
                      onClick={(e) => {
                        updateAttribute(attr.id, "forEach", true);
                        e.currentTarget.blur();
                      }}
                      className={`${formStyles.forEachToggle} ${attr.forEach ? formStyles.forEachToggleActive : ""}`}
                      title="Execute once per array element"
                    >
                      <Layers className={styles.iconSm} />
                      <span>Multi</span>
                    </button>
                  </div>
                )}
              <button
                onClick={() => removeAttribute(attr.id)}
                className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
                title="Remove attribute"
              >
                <Trash2 className={styles.iconSm} />
              </button>
            </div>
            {attr.validationError && (
              <div className={formStyles.attrValidationError}>
                {attr.validationError}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

const HttpConfiguration: React.FC = () => {
  const {
    endpoint,
    httpTimeout,
    healthCheck,
    setEndpoint,
    setHttpTimeout,
    setHealthCheck,
  } = useStepEditingContext();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>HTTP Configuration</label>
      </div>
      <div className={formStyles.httpFields}>
        <div className={formStyles.row}>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.label}>Endpoint URL</label>
            <input
              type="text"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder="http://localhost:8080/process"
              className={formStyles.formControl}
            />
          </div>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>Timeout</label>
            <DurationInput value={httpTimeout} onChange={setHttpTimeout} />
          </div>
        </div>
        <div className={formStyles.field}>
          <label className={formStyles.label}>
            Health Check URL (optional)
          </label>
          <input
            type="text"
            value={healthCheck}
            onChange={(e) => setHealthCheck(e.target.value)}
            placeholder="http://localhost:8080/health"
            className={formStyles.formControl}
          />
        </div>
      </div>
    </div>
  );
};

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
}) => {
  const {
    stepId,
    stepType,
    predicate,
    setPredicate,
    predicateLanguage,
    setPredicateLanguage,
    script,
    setScript,
    scriptLanguage,
    setScriptLanguage,
    saving,
    error,
    handleSave,
    isCreateMode,
    contextValue,
  } = useStepEditorForm(step, onUpdate, onClose);

  const { dimensions, mounted } = useModalDimensions(diagramContainerRef);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [onClose]);

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  if (!mounted) return null;

  const modalContent = (
    <StepEditingContext.Provider value={contextValue}>
      <div className={styles.backdrop} onClick={handleBackdropClick}>
        <div
          className={styles.content}
          style={{
            width: `${dimensions.width}px`,
            minHeight: `${dimensions.minHeight}px`,
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <div className={styles.header}>
            <h2 className={styles.title}>
              {isCreateMode ? "Create New Step" : `Edit Step: ${stepId}`}
            </h2>
          </div>

          <div className={styles.body}>
            <div className={formStyles.formContainer}>
              {/* Basic Fields */}
              <BasicFields />

              {/* Unified Attributes Section */}
              <AttributesSection />

              {/* Predicate */}
              <ScriptConfigEditor
                label="Predicate (Optional)"
                value={predicate}
                onChange={setPredicate}
                language={predicateLanguage}
                onLanguageChange={setPredicateLanguage}
                containerClassName={formStyles.predicateEditorContainer}
              />

              {/* Type-Specific Configuration */}
              {stepType === "script" ? (
                <ScriptConfigEditor
                  label="Script Code"
                  value={script}
                  onChange={setScript}
                  language={scriptLanguage}
                  onLanguageChange={setScriptLanguage}
                  containerClassName={formStyles.scriptEditorContainer}
                />
              ) : (
                <HttpConfiguration />
              )}

              {error && <div className={formStyles.errorMessage}>{error}</div>}
            </div>
          </div>

          <div className={styles.footer}>
            <button
              onClick={onClose}
              disabled={saving}
              className={styles.buttonCancel}
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className={styles.buttonSave}
            >
              {saving
                ? isCreateMode
                  ? "Creating..."
                  : "Saving..."
                : isCreateMode
                  ? "Create"
                  : "Save"}
            </button>
          </div>
        </div>
      </div>
    </StepEditingContext.Provider>
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
