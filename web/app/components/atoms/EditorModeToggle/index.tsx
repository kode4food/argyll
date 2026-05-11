import React from "react";

interface EditorModeToggleProps {
  editorMode: "basic" | "json";
  onChange: (mode: "basic" | "json") => void;
  basicLabel: string;
  jsonLabel: string;
  groupClassName: string;
  buttonClassName: string;
  activeClassName: string;
}

const EditorModeToggle: React.FC<EditorModeToggleProps> = ({
  editorMode,
  onChange,
  basicLabel,
  jsonLabel,
  groupClassName,
  buttonClassName,
  activeClassName,
}) => (
  <div className={groupClassName}>
    <button
      type="button"
      className={`${buttonClassName} ${editorMode === "basic" ? activeClassName : ""}`}
      onClick={() => onChange("basic")}
    >
      {basicLabel}
    </button>
    <button
      type="button"
      className={`${buttonClassName} ${editorMode === "json" ? activeClassName : ""}`}
      onClick={() => onChange("json")}
    >
      {jsonLabel}
    </button>
  </div>
);

export default EditorModeToggle;
