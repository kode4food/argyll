import {
  AttributeType,
  InputCollect,
  SCRIPT_LANGUAGE_ARGYLL,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";
import { AttributeRoleType } from "./stepEditorTypes";

export const attributeTypes: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

export const attributeRoleTypes: AttributeRoleType[] = [
  "required",
  "optional",
  "const",
  "meta",
  "output",
];

export const inputCollectTypes: InputCollect[] = [
  "first",
  "last",
  "some",
  "all",
  "none",
];

export const predicateLanguageOptions = [
  { value: SCRIPT_LANGUAGE_ARGYLL, labelKey: "script.language.argyll" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
];

const mappingScriptPlaceholderKeys: Record<string, string> = {
  [SCRIPT_LANGUAGE_ARGYLL]: "stepEditor.mappingScriptPlaceholderArgyll",
  [SCRIPT_LANGUAGE_LUA]: "stepEditor.mappingScriptPlaceholderLua",
};

const matchScriptPlaceholderKeys: Record<string, string> = {
  [SCRIPT_LANGUAGE_ARGYLL]: "stepEditor.matchScriptPlaceholderArgyll",
  [SCRIPT_LANGUAGE_LUA]: "stepEditor.matchScriptPlaceholderLua",
};

export const getMappingScriptPlaceholderKey = (language?: string): string => {
  if (!language) {
    return mappingScriptPlaceholderKeys[SCRIPT_LANGUAGE_LUA];
  }

  return (
    mappingScriptPlaceholderKeys[language] ||
    mappingScriptPlaceholderKeys[SCRIPT_LANGUAGE_LUA]
  );
};

export const getMatchScriptPlaceholderKey = (language?: string): string => {
  if (!language) {
    return matchScriptPlaceholderKeys[SCRIPT_LANGUAGE_ARGYLL];
  }

  return (
    matchScriptPlaceholderKeys[language] ||
    matchScriptPlaceholderKeys[SCRIPT_LANGUAGE_ARGYLL]
  );
};
