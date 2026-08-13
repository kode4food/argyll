import React, { useMemo } from "react";
import CodeMirror from "@uiw/react-codemirror";
import { codemirrorBasicSetup } from "@/utils/codemirrorSetup";
import { StreamLanguage } from "@codemirror/language";
import { json as jsonLanguage } from "@codemirror/lang-json";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { scheme } from "@codemirror/legacy-modes/mode/scheme";
import { EditorView } from "@codemirror/view";
import styles from "./ScriptEditor.module.css";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  language?: string;
  height?: string;
  theme?: "light" | "dark";
}

const ScriptEditor: React.FC<ScriptEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  language = "lua",
  height = "100%",
  theme = "dark",
}) => {
  const extensions = useMemo(() => {
    if (language === "json") {
      return [jsonLanguage(), EditorView.lineWrapping];
    }
    const langMode = language === "lua" ? lua : scheme;
    const langExt = StreamLanguage.define(langMode);
    return [langExt, EditorView.lineWrapping];
  }, [language]);

  return (
    <div className={styles.editor}>
      <CodeMirror
        value={value}
        height={height}
        className={styles.codemirror}
        extensions={extensions}
        onChange={onChange}
        readOnly={readOnly}
        theme={theme}
        basicSetup={codemirrorBasicSetup}
      />
    </div>
  );
};

export default ScriptEditor;
