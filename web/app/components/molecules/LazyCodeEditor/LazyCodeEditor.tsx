import React, { Suspense, lazy } from "react";
import styles from "./LazyCodeEditor.module.css";
import { useTheme } from "@/app/store/themeStore";
import { CODEMIRROR_BASIC_SETUP } from "@/utils/codemirrorSetup";

const CodeMirror = lazy(() => import("@uiw/react-codemirror"));

interface LazyCodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  height?: string;
}

const EditorFallback = () => <div className={styles.fallback} />;

const LazyCodeEditor: React.FC<LazyCodeEditorProps> = ({
  value,
  onChange,
  height = "100%",
}) => (
  <Suspense fallback={<EditorFallback />}>
    <CodeMirrorEditor value={value} onChange={onChange} height={height} />
  </Suspense>
);

const CodeMirrorEditor: React.FC<LazyCodeEditorProps> = ({
  value,
  onChange,
  height,
}) => {
  const theme = useTheme();
  const [json, setJson] = React.useState<any>(null);
  const [EditorView, setEditorView] = React.useState<any>(null);

  React.useEffect(() => {
    Promise.all([
      import("@codemirror/lang-json"),
      import("@codemirror/view"),
    ]).then(([jsonModule, viewModule]) => {
      setJson(() => jsonModule.json);
      setEditorView(() => viewModule.EditorView);
    });
  }, []);

  if (!json || !EditorView) {
    return <EditorFallback />;
  }

  return (
    <CodeMirror
      value={value}
      height={height}
      className={styles.codemirror}
      extensions={[json(), EditorView.lineWrapping]}
      onChange={onChange}
      theme={theme}
      basicSetup={CODEMIRROR_BASIC_SETUP}
    />
  );
};

export default LazyCodeEditor;
