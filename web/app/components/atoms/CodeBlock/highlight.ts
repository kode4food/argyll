import { StreamLanguage, type StreamParser } from "@codemirror/language";
import { go } from "@codemirror/legacy-modes/mode/go";
import {
  javascript,
  json,
  typescript,
} from "@codemirror/legacy-modes/mode/javascript";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { python } from "@codemirror/legacy-modes/mode/python";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { yaml } from "@codemirror/legacy-modes/mode/yaml";
import { classHighlighter, highlightCode, tags } from "@lezer/highlight";

export interface CodeSegment {
  text: string;
  className: string;
}

const MODES: Record<string, StreamParser<unknown>> = {
  argex: javascript,
  argyll: javascript,
  bash: shell,
  go,
  javascript,
  js: javascript,
  json,
  lua,
  py: python,
  python,
  sh: shell,
  shell,
  ts: typescript,
  typescript,
  yaml,
  yml: yaml,
};

const languages = new Map<StreamParser<unknown>, StreamLanguage<unknown>>();

const languageFor = (name: string) => {
  const mode = MODES[name.toLowerCase()];
  if (!mode) return undefined;
  const cached = languages.get(mode);
  if (cached) return cached;
  // Legacy modes emit "property", which has no default tag
  const language = StreamLanguage.define({
    ...mode,
    tokenTable: { property: tags.propertyName, ...mode.tokenTable },
  });
  languages.set(mode, language);
  return language;
};

// Splits code into runs of text tagged with `tok-*` classes. Unknown languages
// come back as one unstyled run
export const highlight = (code: string, language: string): CodeSegment[] => {
  const lang = languageFor(language);
  if (!lang) return [{ text: code, className: "" }];
  const res: CodeSegment[] = [];
  highlightCode(
    code,
    lang.parser.parse(code),
    classHighlighter,
    (text, className) => res.push({ text, className }),
    () => res.push({ text: "\n", className: "" })
  );
  return res;
};
