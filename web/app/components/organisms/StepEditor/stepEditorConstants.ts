import {
  AttributeType,
  InputCollect,
  SCRIPT_LANGUAGE_ALE,
  SCRIPT_LANGUAGE_JPATH,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";

export const ATTRIBUTE_TYPES: AttributeType[] = [
  AttributeType.String,
  AttributeType.Number,
  AttributeType.Boolean,
  AttributeType.Object,
  AttributeType.Array,
  AttributeType.Any,
];

export const INPUT_COLLECT_TYPES: InputCollect[] = [
  "first",
  "last",
  "some",
  "all",
  "none",
];

export const PREDICATE_LANGUAGE_OPTIONS = [
  { value: SCRIPT_LANGUAGE_ALE, labelKey: "script.language.ale" },
  { value: SCRIPT_LANGUAGE_JPATH, labelKey: "script.language.jpath" },
  { value: SCRIPT_LANGUAGE_LUA, labelKey: "script.language.lua" },
];

const MAPPING_SCRIPT_PLACEHOLDER_KEYS: Record<string, string> = {
  [SCRIPT_LANGUAGE_ALE]: "stepEditor.mappingScriptPlaceholderAle",
  [SCRIPT_LANGUAGE_JPATH]: "stepEditor.mappingScriptPlaceholderJPath",
  [SCRIPT_LANGUAGE_LUA]: "stepEditor.mappingScriptPlaceholderLua",
};

const MATCH_SCRIPT_PLACEHOLDER_KEYS: Record<string, string> = {
  [SCRIPT_LANGUAGE_ALE]: "stepEditor.matchScriptPlaceholderAle",
  [SCRIPT_LANGUAGE_JPATH]: "stepEditor.matchScriptPlaceholderJPath",
  [SCRIPT_LANGUAGE_LUA]: "stepEditor.matchScriptPlaceholderLua",
};

export const getMappingScriptPlaceholderKey = (language?: string): string => {
  if (!language) {
    return MAPPING_SCRIPT_PLACEHOLDER_KEYS[SCRIPT_LANGUAGE_LUA];
  }

  return (
    MAPPING_SCRIPT_PLACEHOLDER_KEYS[language] ||
    MAPPING_SCRIPT_PLACEHOLDER_KEYS[SCRIPT_LANGUAGE_LUA]
  );
};

export const getMatchScriptPlaceholderKey = (language?: string): string => {
  if (!language) {
    return MATCH_SCRIPT_PLACEHOLDER_KEYS[SCRIPT_LANGUAGE_JPATH];
  }

  return (
    MATCH_SCRIPT_PLACEHOLDER_KEYS[language] ||
    MATCH_SCRIPT_PLACEHOLDER_KEYS[SCRIPT_LANGUAGE_JPATH]
  );
};
