import {
  Step,
  AttributeSpec,
  AttributeType,
  AttributeRole,
  StepType,
} from "@/app/api";
import { getSortedAttributes, validateDefaultValue } from "@/utils/stepUtils";
import { getArgIcon } from "@/utils/iconRegistry";

export type AttributeRoleType = "input" | "optional" | "const" | "output";

export interface Attribute {
  id: string;
  attrType: AttributeRoleType;
  name: string;
  dataType: AttributeType;
  defaultValue?: string;
  forEach?: boolean;
  flowMap?: string;
  validationError?: string;
}

export interface ValidationError {
  key: string;
  vars?: Record<string, string>;
}

export function buildAttributesFromStep(step: Step | null): Attribute[] {
  if (!step) return [];

  const timestamp = Date.now();
  const flowConfig = step.flow;

  return getSortedAttributes(step.attributes || {}).map(
    ({ name, spec }, index) => {
      const attrType =
        spec.role === AttributeRole.Required
          ? "input"
          : spec.role === AttributeRole.Optional
            ? "optional"
            : spec.role === AttributeRole.Const
              ? "const"
              : ("output" as AttributeRoleType);
      const prefix =
        spec.role === AttributeRole.Output
          ? "output"
          : spec.role === AttributeRole.Const
            ? "const"
            : "input";

      let flowMap: string | undefined;
      if (flowConfig) {
        if (spec.role === AttributeRole.Output && flowConfig.output_map) {
          for (const [childName, outputName] of Object.entries(
            flowConfig.output_map
          )) {
            if (outputName === name) {
              flowMap = childName;
              break;
            }
          }
        } else if (flowConfig.input_map) {
          flowMap = flowConfig.input_map[name] ?? undefined;
        }
      }

      return {
        id: `${prefix}-${index}-${timestamp}`,
        attrType,
        name,
        dataType: spec.type || AttributeType.String,
        defaultValue:
          spec.role === AttributeRole.Optional ||
          spec.role === AttributeRole.Const
            ? spec.default !== undefined
              ? String(spec.default)
              : undefined
            : undefined,
        forEach: spec.for_each || false,
        flowMap,
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

    if (
      (attr.attrType === "optional" || attr.attrType === "const") &&
      attr.defaultValue
    ) {
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

    if (attr.attrType === "const" && !attr.defaultValue?.trim()) {
      return {
        key: "stepEditor.constDefaultRequired",
        vars: { name: attr.name },
      };
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
          : a.attrType === "const"
            ? AttributeRole.Const
            : AttributeRole.Output;

    const spec: AttributeSpec = {
      role,
      type: a.dataType,
    };

    if (
      (a.attrType === "optional" || a.attrType === "const") &&
      a.defaultValue?.trim()
    ) {
      spec.default = a.defaultValue.trim();
    }

    if (a.forEach) {
      spec.for_each = true;
    }

    stepAttributes[a.name] = spec;
  });
  return stepAttributes;
}

export function buildFlowMaps(attributes: Attribute[]): {
  inputMap: Record<string, string>;
  outputMap: Record<string, string>;
} {
  const inputMap: Record<string, string> = {};
  const outputMap: Record<string, string> = {};

  attributes.forEach((attr) => {
    if (!attr.flowMap?.trim() || !attr.name.trim()) {
      return;
    }
    if (attr.attrType === "output") {
      outputMap[attr.flowMap.trim()] = attr.name;
      return;
    }
    inputMap[attr.name] = attr.flowMap.trim();
  });

  return { inputMap, outputMap };
}

export function getValidationError({
  isCreateMode,
  stepId,
  attributes,
  stepType,
  script,
  endpoint,
  httpTimeout,
  flowGoals,
}: {
  isCreateMode: boolean;
  stepId: string;
  attributes: Attribute[];
  stepType: StepType;
  script: string;
  endpoint: string;
  httpTimeout: number;
  flowGoals: string;
}): ValidationError | null {
  if (isCreateMode && !stepId.trim()) {
    return { key: "stepEditor.stepIdRequired" };
  }

  const attrError = validateAttributesList(attributes);
  if (attrError) {
    return attrError;
  }

  if (stepType === "flow") {
    if (parseFlowGoals(flowGoals).length === 0) {
      return { key: "stepEditor.flowGoalsRequired" };
    }
    const flowMapError = validateFlowMaps(attributes);
    if (flowMapError) {
      return flowMapError;
    }
    return null;
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

function validateFlowMaps(attributes: Attribute[]): ValidationError | null {
  const inputTargets = new Set<string>();
  const outputTargets = new Set<string>();

  for (const attr of attributes) {
    if (!attr.flowMap?.trim()) {
      continue;
    }
    const target = attr.flowMap.trim();
    if (attr.attrType === "output") {
      if (outputTargets.has(target)) {
        return {
          key: "stepEditor.duplicateFlowOutputMap",
          vars: { name: target },
        };
      }
      outputTargets.add(target);
      continue;
    }

    if (inputTargets.has(target)) {
      return {
        key: "stepEditor.duplicateFlowInputMap",
        vars: { name: target },
      };
    }
    inputTargets.add(target);
  }

  return null;
}

export function parseFlowGoals(value: string): string[] {
  return value
    .split(/[\n,]+/)
    .map((goal) => goal.trim())
    .filter((goal) => goal.length > 0);
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
  flowGoals,
  flowInputMap,
  flowOutputMap,
  memoizable,
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
  flowGoals: string;
  flowInputMap: Record<string, string>;
  flowOutputMap: Record<string, string>;
  memoizable: boolean;
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
    memoizable,
  };

  if (stepType === "flow") {
    stepData.flow = {
      goals: parseFlowGoals(flowGoals),
      input_map: flowInputMap,
      output_map: flowOutputMap,
    };
    stepData.http = undefined;
    stepData.script = undefined;
  } else if (stepType === "script") {
    stepData.script = {
      language: scriptLanguage,
      script: script.trim(),
    };
    stepData.http = undefined;
    stepData.flow = undefined;
  } else {
    stepData.http = {
      endpoint: endpoint.trim(),
      health_check: healthCheck.trim() || undefined,
      timeout: httpTimeout,
    };
    stepData.script = undefined;
    stepData.flow = undefined;
  }

  return stepData;
}
