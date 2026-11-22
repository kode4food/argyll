import { useMemo } from "react";
import { StreamLanguage } from "@codemirror/language";
import { json as jsonLanguage } from "@codemirror/lang-json";
import { markdown as markdownLanguage } from "@codemirror/lang-markdown";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { javascript } from "@codemirror/legacy-modes/mode/javascript";
import { EditorView } from "@codemirror/view";

export const useScriptExtensions = (
  language: string,
  completions?: readonly string[]
) =>
  useMemo(() => {
    if (language === "json") {
      return [jsonLanguage(), EditorView.lineWrapping];
    }
    if (language === "markdown") {
      return [markdownLanguage(), EditorView.lineWrapping];
    }
    const langMode = language === "lua" ? lua : javascript;
    const langExt = StreamLanguage.define(langMode);
    return [
      langExt,
      ...(completions?.length
        ? [langExt.data.of({ autocomplete: completions })]
        : []),
      EditorView.lineWrapping,
    ];
  }, [completions, language]);
