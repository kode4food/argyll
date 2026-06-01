import React from "react";
import SegmentedControl from "@/app/components/atoms/SegmentedControl";

interface EditorModeToggleProps {
  editorMode: "basic" | "json";
  onChange: (mode: "basic" | "json") => void;
  basicLabel: string;
  jsonLabel: string;
}

const EditorModeToggle: React.FC<EditorModeToggleProps> = ({
  editorMode,
  onChange,
  basicLabel,
  jsonLabel,
}) => (
  <SegmentedControl
    options={[
      { value: "basic", label: basicLabel },
      { value: "json", label: jsonLabel },
    ]}
    value={editorMode}
    onChange={(v) => onChange(v as "basic" | "json")}
  />
);

export default EditorModeToggle;
