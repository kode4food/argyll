import React, { useMemo } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { StreamLanguage } from "@codemirror/language";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { scheme } from "@codemirror/legacy-modes/mode/scheme";
import { json } from "@codemirror/legacy-modes/mode/javascript";
import { EditorView } from "@codemirror/view";
import styles from "./ScriptEditor.module.css";

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
    if (language === "json") {
      return [StreamLanguage.define(json), EditorView.lineWrapping];
    }
    const langMode = language === "lua" ? lua : scheme;
    const langExt = StreamLanguage.define(langMode);
    return [langExt, EditorView.lineWrapping];
  }, [language]);

  return (
    <div className={styles.editor}>
      <CodeMirror
        value={value}
        className={styles.codemirror}
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
