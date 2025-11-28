"use client";

import React, { useState, useEffect, useCallback } from "react";
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
import {
  Step,
  AttributeSpec,
  AttributeType,
  AttributeRole,
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_LUA,
  StepType,
} from "@/app/api";
import { SpudsApi } from "@/app/api";
import ScriptConfigEditor from "../molecules/ScriptConfigEditor";
import DurationInput from "../molecules/DurationInput";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import { getArgIcon } from "@/utils/argIcons";
import { getSortedAttributes, validateDefaultValue } from "@/utils/stepUtils";

interface StepEditorProps {
  step: Step | null;
  onClose: () => void;
  onUpdate: (updatedStep: Step) => void;
  diagramContainerRef?: React.RefObject<HTMLDivElement>;
}

type AttributeRoleType = "input" | "optional" | "output";

interface Attribute {
  id: string;
  attrType: AttributeRoleType;
  name: string;
  dataType: AttributeType;
  defaultValue?: string;
  forEach?: boolean;
  validationError?: string;
}

const ATTRIBUTE_TYPES: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

const StepEditor: React.FC<StepEditorProps> = ({
  step,
  onClose,
  onUpdate,
  diagramContainerRef,
}) => {
  const isCreateMode = step === null;
  const [stepId, setStepId] = useState(step?.id || "");
  const [name, setName] = useState(step?.name || "");
  const [stepType, setStepType] = useState<StepType>(step?.type || "sync");
  const [predicate, setPredicate] = useState(step?.predicate?.script || "");
  const [predicateLanguage, setPredicateLanguage] = useState(
    step?.predicate?.language || SCRIPT_LANGUAGE_LUA
  );

  // HTTP config state
  const [endpoint, setEndpoint] = useState(step?.http?.endpoint || "");
  const [healthCheck, setHealthCheck] = useState(
    step?.http?.health_check || ""
  );
  const [httpTimeout, setHttpTimeout] = useState(
    step && step.type !== "script" && step.http?.timeout
      ? step.http.timeout
      : 5000
  );

  // Script config state
  const [script, setScript] = useState(step?.script?.script || "");
  const [scriptLanguage, setScriptLanguage] = useState(
    step?.script?.language || SCRIPT_LANGUAGE_LUA
  );

  const [attributes, setAttributes] = useState<Attribute[]>(() => {
    if (!step) return [];

    const timestamp = Date.now();

    return getSortedAttributes(step.attributes || {}).map(
      ({ name, spec }, index) => {
        const attrType =
          spec.role === AttributeRole.Required
            ? "input"
            : spec.role === AttributeRole.Optional
              ? "optional"
              : ("output" as AttributeRoleType);
        const prefix = spec.role === AttributeRole.Output ? "output" : "input";

        return {
          id: `${prefix}-${index}-${timestamp}`,
          attrType,
          name,
          dataType: spec.type || AttributeType.String,
          defaultValue:
            spec.role === AttributeRole.Optional && spec.default !== undefined
              ? String(spec.default)
              : undefined,
          forEach: spec.for_each || false,
        };
      }
    );
  });

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  const getDimensions = useCallback(() => {
    if (diagramContainerRef?.current) {
      const rect = diagramContainerRef.current.getBoundingClientRect();
      return {
        width: Math.min(rect.width * 0.8, 1000),
        minHeight: rect.height * 0.9,
      };
    }
    return { width: 1000, minHeight: 800 };
  }, [diagramContainerRef]);

  const [dimensions, setDimensions] = useState(getDimensions);

  useEffect(() => {
    setMounted(true);
    const dims = getDimensions();
    setDimensions(dims);
  }, [getDimensions]);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [onClose]);

  const validateAttributes = (): string | null => {
    const names = new Set<string>();
    for (const attr of attributes) {
      if (!attr.name.trim()) {
        return "All attribute names are required";
      }
      if (names.has(attr.name)) {
        return `Duplicate attribute name: ${attr.name}`;
      }
      names.add(attr.name);

      if (attr.attrType === "optional" && attr.defaultValue) {
        const validation = validateDefaultValue(
          attr.defaultValue,
          attr.dataType
        );
        if (!validation.valid) {
          return `Invalid default value for "${attr.name}": ${validation.error}`;
        }
      }
    }
    return null;
  };

  const handleSave = async () => {
    // Validate step ID for create mode
    if (isCreateMode && !stepId.trim()) {
      setError("Step ID is required");
      return;
    }

    // Validate attributes
    const attrError = validateAttributes();
    if (attrError) {
      setError(attrError);
      return;
    }

    // Type-specific validation
    if (stepType === "script") {
      if (!script.trim()) {
        setError("Script code is required");
        return;
      }
    } else {
      if (!endpoint.trim()) {
        setError("HTTP endpoint is required");
        return;
      }
      if (!httpTimeout || httpTimeout <= 0) {
        setError("Timeout must be a positive number");
        return;
      }
    }

    setSaving(true);
    setError(null);

    try {
      const api = new SpudsApi();

      // Convert attributes back to step format
      const stepAttributes: Record<string, AttributeSpec> = {};
      attributes.forEach((a) => {
        const role =
          a.attrType === "input"
            ? AttributeRole.Required
            : a.attrType === "optional"
              ? AttributeRole.Optional
              : AttributeRole.Output;

        const spec: AttributeSpec = {
          role,
          type: a.dataType,
        };

        if (a.attrType === "optional" && a.defaultValue?.trim()) {
          spec.default = a.defaultValue.trim();
        }

        if (a.forEach) {
          spec.for_each = true;
        }

        stepAttributes[a.name] = spec;
      });

      const stepData: Step = {
        id: stepId.trim(),
        name,
        type: stepType,
        version: step?.version || "1.0.0",
        attributes: stepAttributes,
        predicate: predicate.trim()
          ? {
              language: predicateLanguage,
              script: predicate.trim(),
            }
          : undefined,
      };

      // Add type-specific config
      if (stepType === "script") {
        stepData.script = {
          language: scriptLanguage,
          script: script.trim(),
        };
        stepData.http = undefined;
      } else {
        stepData.http = {
          endpoint: endpoint.trim(),
          health_check: healthCheck.trim() || undefined,
          timeout: httpTimeout,
        };
        stepData.script = undefined;
      }

      let resultStep: Step;
      if (isCreateMode) {
        resultStep = await api.registerStep(stepData);
      } else {
        resultStep = await api.updateStep(stepId, stepData);
      }

      onUpdate(resultStep);
      onClose();
    } catch (err: any) {
      setError(err.response?.data?.error || err.message || "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const addAttribute = () => {
    setAttributes([
      ...attributes,
      {
        id: `attr-${Date.now()}`,
        attrType: "input",
        name: "",
        dataType: AttributeType.String,
      },
    ]);
  };

  const updateAttribute = (id: string, field: keyof Attribute, value: any) => {
    setAttributes(
      attributes.map((attr) => {
        if (attr.id !== id) return attr;

        const updated = { ...attr, [field]: value };

        if (
          (field === "defaultValue" || field === "dataType") &&
          updated.attrType === "optional" &&
          updated.defaultValue
        ) {
          const validation = validateDefaultValue(
            updated.defaultValue,
            updated.dataType
          );
          updated.validationError = validation.valid
            ? undefined
            : validation.error;
        }

        if (field === "attrType" && value !== "optional") {
          updated.validationError = undefined;
        }

        return updated;
      })
    );
  };

  const removeAttribute = (id: string) => {
    setAttributes(attributes.filter((attr) => attr.id !== id));
  };

  const getAttributeIcon = (attrType: AttributeRoleType) => {
    const argType = attrType === "input" ? "required" : attrType;
    const { Icon, className } = getArgIcon(argType);
    return <Icon size={16} className={className} />;
  };

  const cycleAttributeType = (id: string, currentType: AttributeRoleType) => {
    const types: AttributeRoleType[] = ["input", "optional", "output"];
    const currentIndex = types.indexOf(currentType);
    const nextIndex = (currentIndex + 1) % types.length;
    updateAttribute(id, "attrType", types[nextIndex]);
  };

  if (!mounted) return null;

  const modalContent = (
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
                      <Icon size={16} />
                      <span>{label}</span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Unified Attributes Section */}
            <div className={formStyles.section}>
              <div className={formStyles.sectionHeader}>
                <label className={formStyles.label}>Attributes</label>
                <button
                  onClick={addAttribute}
                  className={`${formStyles.iconButton} ${formStyles.addButtonStyle}`}
                  title="Add attribute"
                >
                  <Plus size={16} />
                </button>
              </div>
              <div className={formStyles.argList}>
                {attributes.map((attr) => (
                  <div key={attr.id} className={formStyles.attrRow}>
                    <div className={formStyles.attrRowInputs}>
                      <button
                        type="button"
                        onClick={() =>
                          cycleAttributeType(attr.id, attr.attrType)
                        }
                        className={`${formStyles.iconButton} ${formStyles.attrIconButtonStyle}`}
                        title={`Click to cycle type (current: ${attr.attrType})`}
                      >
                        {getAttributeIcon(attr.attrType)}
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
                            updateAttribute(
                              attr.id,
                              "defaultValue",
                              e.target.value
                            )
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
                              <Square size={14} />
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
                              <Layers size={14} />
                              <span>Multi</span>
                            </button>
                          </div>
                        )}
                      <button
                        onClick={() => removeAttribute(attr.id)}
                        className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
                        title="Remove attribute"
                      >
                        <Trash2 size={14} />
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
                      <DurationInput
                        value={httpTimeout}
                        onChange={setHttpTimeout}
                      />
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
  );

  return createPortal(modalContent, document.body);
};

export default StepEditor;
