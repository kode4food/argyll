import { AttributeType } from "@/app/api";

export interface FlowInputMeta {
  hasRequiredSpec: boolean;
  hasSpecDefault: boolean;
  explicitDefault?: string;
  hasConflictingDefaults: boolean;
}

const TYPE_DEFAULT_MAP: Partial<Record<AttributeType, string>> = {
  [AttributeType.Number]: "0",
  [AttributeType.Boolean]: "false",
  [AttributeType.Object]: "{}",
  [AttributeType.Array]: "[]",
  [AttributeType.Null]: "null",
};

export function getTypeDefaultValue(
  attrType?: AttributeType
): string | undefined {
  if (!attrType) return undefined;
  return TYPE_DEFAULT_MAP[attrType];
}

export function mergeInputType(
  existingType: AttributeType | undefined,
  nextType: AttributeType | undefined
): AttributeType | undefined {
  if (!existingType) {
    return nextType;
  }
  if (!nextType || existingType === nextType) {
    return existingType;
  }
  return AttributeType.Any;
}

export function normalizeDefaultValue(
  defaultValue?: string
): string | undefined {
  if (defaultValue === undefined) {
    return undefined;
  }
  const trimmed = defaultValue.trim();
  if (!trimmed) {
    return "";
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (parsed === null || parsed === undefined) {
      return "";
    }
    if (typeof parsed === "string") {
      return parsed;
    }
    return JSON.stringify(parsed);
  } catch {
    return trimmed;
  }
}

export function getOrCreateFlowInputMeta(
  inputMetaMap: Map<string, FlowInputMeta>,
  name: string
): FlowInputMeta {
  const existing = inputMetaMap.get(name);
  if (existing) {
    return existing;
  }

  const created: FlowInputMeta = {
    hasRequiredSpec: false,
    hasSpecDefault: false,
    hasConflictingDefaults: false,
  };
  inputMetaMap.set(name, created);
  return created;
}

export function mergeExplicitDefault(
  meta: FlowInputMeta,
  normalizedDefault: string
): void {
  if (meta.hasConflictingDefaults) {
    return;
  }
  if (meta.explicitDefault === undefined) {
    meta.explicitDefault = normalizedDefault;
    return;
  }
  if (meta.explicitDefault !== normalizedDefault) {
    meta.hasConflictingDefaults = true;
    meta.explicitDefault = undefined;
  }
}

export function getInputPriorityRank(
  option: { name: string; required?: boolean; unreachable?: boolean },
  outputSet: Set<string>,
  inputMetaMap: Map<string, FlowInputMeta>
): number {
  if (option.unreachable) {
    return 4;
  }
  const meta = inputMetaMap.get(option.name);
  if (outputSet.has(option.name)) {
    return 3;
  }
  if (option.required || meta?.hasRequiredSpec) {
    return 0;
  }
  if (meta?.hasSpecDefault) {
    return 2;
  }
  return 1;
}
