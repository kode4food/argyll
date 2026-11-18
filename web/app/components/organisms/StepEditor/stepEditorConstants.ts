import {
  AgentConfig,
  InputCollect,
  SCRIPT_LANGUAGE_ARGYLL,
  SCRIPT_LANGUAGE_LUA,
} from "@/app/api";
import { AttributeRoleType } from "./stepEditorTypes";

export const DEFAULT_AGENT_MAX_TURNS = 4;
export const DEFAULT_AGENT_MAX_TOKENS = 1024;

export const EMPTY_AGENT: AgentConfig = {
  provider: "",
  model: "",
  prologue: "",
  max_turns: DEFAULT_AGENT_MAX_TURNS,
  max_tokens: DEFAULT_AGENT_MAX_TOKENS,
};

export const attributeRoleTypes: AttributeRoleType[] = [
  "required",
  "optional",
  "const",
  "meta",
  "feed",
  "output",
  "metric",
];

export const inputCollectTypes: InputCollect[] = [
  "first",
  "last",
  "some",
  "all",
  "none",
];

export const SCRIPT_LANGUAGE_OPTIONS = [
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
    return mappingScriptPlaceholderKeys[SCRIPT_LANGUAGE_ARGYLL];
  }

  return (
    mappingScriptPlaceholderKeys[language] ||
    mappingScriptPlaceholderKeys[SCRIPT_LANGUAGE_ARGYLL]
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
