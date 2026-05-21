import { AttributeType, InputCollect } from "@/app/api";

export type AttributeRoleType =
  | "required"
  | "optional"
  | "const"
  | "meta"
  | "output";

export interface Attribute {
  id: string;
  role: AttributeRoleType;
  name: string;
  dataType: AttributeType;
  collect?: InputCollect;
  defaultValue?: string;
  deadline?: number;
  forEach?: boolean;
  matchLanguage?: string;
  matchScript?: string;
  metaKey?: string;
  mappingName?: string;
  mappingLanguage?: string;
  mappingScript?: string;
  validationError?: string;
}

export interface ValidationError {
  key: string;
  vars?: Record<string, string>;
}

export interface AttributeIndex {
  index: number;
  timestamp: number;
}
