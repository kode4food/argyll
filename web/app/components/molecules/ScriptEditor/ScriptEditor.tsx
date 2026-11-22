import React from "react";
import CodeMirror from "@uiw/react-codemirror";
import { CODEMIRROR_BASIC_SETUP } from "@/utils/codemirrorSetup";
import styles from "./ScriptEditor.module.css";
import { useScriptExtensions } from "./useScriptExtensions";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  completions?: readonly string[];
  readOnly?: boolean;
  language?: string;
  height?: string;
  theme?: "light" | "dark";
}

const ScriptEditor: React.FC<ScriptEditorProps> = ({
  value,
  onChange,
  completions,
  readOnly = false,
  language = "lua",
  height = "100%",
  theme = "dark",
}) => {
  const extensions = useScriptExtensions(language, completions);

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
        basicSetup={CODEMIRROR_BASIC_SETUP}
      />
    </div>
  );
};

export default ScriptEditor;
