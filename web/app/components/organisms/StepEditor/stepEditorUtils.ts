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

export function validateAttributesList(attributes: Attribute[]): string | null {
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
      const validation = validateDefaultValue(attr.defaultValue, attr.dataType);
      if (!validation.valid) {
        return `Invalid default value for "${attr.name}": ${validation.error}`;
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
}): string | null {
  if (isCreateMode && !stepId.trim()) {
    return "Step ID is required";
  }

  const attrError = validateAttributesList(attributes);
  if (attrError) {
    return attrError;
  }

  if (stepType === "script") {
    if (!script.trim()) {
      return "Script code is required";
    }
  } else {
    if (!endpoint.trim()) {
      return "HTTP endpoint is required";
    }
    if (!httpTimeout || httpTimeout <= 0) {
      return "Timeout must be a positive number";
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
