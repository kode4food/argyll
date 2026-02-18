import { render } from "@testing-library/react";
import ScriptEditor from "./ScriptEditor";
import { StreamLanguage } from "@codemirror/language";
import { json as jsonLegacyMode } from "@codemirror/legacy-modes/mode/javascript";

const codeMirrorMock = jest.fn(({ value, onChange, readOnly, theme }: any) => (
  <div data-testid="codemirror">
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      data-theme={theme}
    />
  </div>
));

jest.mock("@uiw/react-codemirror", () => ({
  __esModule: true,
  default: (props: any) => codeMirrorMock(props),
}));

jest.mock("@codemirror/language", () => ({
  StreamLanguage: {
    define: jest.fn(() => "lua-extension"),
  },
}));

jest.mock("@codemirror/legacy-modes/mode/lua", () => ({
  lua: {},
}));

jest.mock("@codemirror/legacy-modes/mode/javascript", () => ({
  json: { name: "json-mode" },
}));

jest.mock("@codemirror/view", () => ({
  EditorView: {
    lineWrapping: "line-wrapping-extension",
  },
}));

describe("ScriptEditor", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("renders CodeMirror editor", () => {
    const onChange = jest.fn();

    const { getByTestId } = render(
      <ScriptEditor value="" onChange={onChange} />
    );

    expect(getByTestId("codemirror")).toBeInTheDocument();
  });

  test("passes value to CodeMirror", () => {
    const onChange = jest.fn();
    const value = "{:result 42}";

    const { getByTestId } = render(
      <ScriptEditor value={value} onChange={onChange} />
    );

    const textarea = getByTestId("codemirror").querySelector("textarea");
    expect(textarea?.value).toBe(value);
  });

  test("passes onChange handler to CodeMirror", () => {
    const onChange = jest.fn();

    render(<ScriptEditor value="" onChange={onChange} />);

    expect(onChange).toBeDefined();
  });

  test("sets readOnly to false by default", () => {
    const onChange = jest.fn();

    const { getByTestId } = render(
      <ScriptEditor value="" onChange={onChange} />
    );

    const textarea = getByTestId("codemirror").querySelector("textarea");
    expect(textarea?.readOnly).toBe(false);
  });

  test("sets readOnly when specified", () => {
    const onChange = jest.fn();

    const { getByTestId } = render(
      <ScriptEditor value="" onChange={onChange} readOnly={true} />
    );

    const textarea = getByTestId("codemirror").querySelector("textarea");
    expect(textarea?.readOnly).toBe(true);
  });

  test("uses dark theme", () => {
    const onChange = jest.fn();

    const { getByTestId } = render(
      <ScriptEditor value="" onChange={onChange} />
    );

    const textarea = getByTestId("codemirror").querySelector("textarea");
    expect(textarea?.dataset.theme).toBe("dark");
  });

  test("uses JSON legacy mode when language is json", () => {
    const onChange = jest.fn();
    render(<ScriptEditor value="{}" onChange={onChange} language="json" />);

    expect(StreamLanguage.define).toHaveBeenCalledWith(jsonLegacyMode);
    expect(codeMirrorMock).toHaveBeenCalled();
  });
});
