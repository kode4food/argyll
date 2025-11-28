"use client";

import React, { useMemo } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { StreamLanguage } from "@codemirror/language";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { EditorView } from "@codemirror/view";
import "../../styles/components/editor.css";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  language?: string;
}

const ScriptEditor: React.FC<ScriptEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  language = "lua",
}) => {
  const extensions = useMemo(() => {
    const langExt =
      language === "lua" ? StreamLanguage.define(lua) : javascript();
    return [langExt, EditorView.lineWrapping];
  }, [language]);

  return (
    <div className="script-editor">
      <CodeMirror
        value={value}
        className="script-editor__codemirror"
        extensions={extensions}
        onChange={onChange}
        readOnly={readOnly}
        theme="dark"
        basicSetup={{
          lineNumbers: true,
          highlightActiveLineGutter: true,
          highlightSpecialChars: true,
          foldGutter: true,
          drawSelection: true,
          dropCursor: true,
          allowMultipleSelections: true,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: true,
          rectangularSelection: true,
          crosshairCursor: true,
          highlightActiveLine: true,
          highlightSelectionMatches: true,
          closeBracketsKeymap: true,
          searchKeymap: true,
          foldKeymap: true,
          completionKeymap: true,
          lintKeymap: true,
        }}
      />
    </div>
  );
};

export default ScriptEditor;
