import {
  Step,
  AttributeSpec,
  AttributeType,
  AttributeRole,
  StepType,
} from "@/app/api";
import { getSortedAttributes, validateDefaultValue } from "@/utils/stepUtils";
import { getArgIcon } from "@/utils/argIcons";

export type AttributeRoleType = "input" | "optional" | "output";

export interface Attribute {
  id: string;
  attrType: AttributeRoleType;
  name: string;
  dataType: AttributeType;
  defaultValue?: string;
  forEach?: boolean;
  validationError?: string;
}

export interface ValidationError {
  key: string;
  vars?: Record<string, string>;
}

export function buildAttributesFromStep(step: Step | null): Attribute[] {
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
}

export function validateAttributesList(
  attributes: Attribute[]
): ValidationError | null {
  const names = new Set<string>();
  for (const attr of attributes) {
    if (!attr.name.trim()) {
      return { key: "stepEditor.attributeNameRequired" };
    }
    if (names.has(attr.name)) {
      return {
        key: "stepEditor.duplicateAttributeName",
        vars: { name: attr.name },
      };
    }
    names.add(attr.name);

    if (attr.attrType === "optional" && attr.defaultValue) {
      const validation = validateDefaultValue(attr.defaultValue, attr.dataType);
      if (!validation.valid) {
        return {
          key: "stepEditor.invalidDefaultValue",
          vars: {
            name: attr.name,
            reason: validation.errorKey ?? "",
          },
        };
      }
    }
  }
  return null;
}

export function getAttributeIconProps(attrType: AttributeRoleType) {
  const argType = attrType === "input" ? "required" : attrType;
  return getArgIcon(argType);
}

export function createStepAttributes(
  attributes: Attribute[]
): Record<string, AttributeSpec> {
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
  return stepAttributes;
}

export function getValidationError({
  isCreateMode,
  stepId,
  attributes,
  stepType,
  script,
  endpoint,
  httpTimeout,
}: {
  isCreateMode: boolean;
  stepId: string;
  attributes: Attribute[];
  stepType: StepType;
  script: string;
  endpoint: string;
  httpTimeout: number;
}): ValidationError | null {
  if (isCreateMode && !stepId.trim()) {
    return { key: "stepEditor.stepIdRequired" };
  }

  const attrError = validateAttributesList(attributes);
  if (attrError) {
    return attrError;
  }

  if (stepType === "script") {
    if (!script.trim()) {
      return { key: "stepEditor.scriptRequired" };
    }
  } else {
    if (!endpoint.trim()) {
      return { key: "stepEditor.endpointRequired" };
    }
    if (!httpTimeout || httpTimeout <= 0) {
      return { key: "stepEditor.timeoutPositive" };
    }
  }

  return null;
}

export function buildStepPayload({
  stepId,
  name,
  stepType,
  attributes,
  predicate,
  predicateLanguage,
  script,
  scriptLanguage,
  endpoint,
  healthCheck,
  httpTimeout,
}: {
  stepId: string;
  name: string;
  stepType: StepType;
  attributes: Record<string, AttributeSpec>;
  predicate: string;
  predicateLanguage: string;
  script: string;
  scriptLanguage: string;
  endpoint: string;
  healthCheck: string;
  httpTimeout: number;
}): Step {
  const stepData: Step = {
    id: stepId.trim(),
    name,
    type: stepType,
    attributes,
    predicate: predicate.trim()
      ? {
          language: predicateLanguage,
          script: predicate.trim(),
        }
      : undefined,
  };

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

  return stepData;
}
