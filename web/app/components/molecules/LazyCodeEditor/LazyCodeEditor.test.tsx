import { render, waitFor } from "@testing-library/react";
import LazyCodeEditor from "./LazyCodeEditor";

const codeMirrorMock = jest.fn(({ theme }: any) => (
  <div data-testid="codemirror" data-theme={theme} />
));

jest.mock("@uiw/react-codemirror", () => ({
  __esModule: true,
  default: (props: any) => codeMirrorMock(props),
}));

jest.mock("@codemirror/lang-json", () => ({
  json: jest.fn(() => "json-extension"),
}));

jest.mock("@codemirror/view", () => ({
  EditorView: {
    lineWrapping: "line-wrapping-extension",
  },
}));

jest.mock("@/app/store/themeStore", () => ({
  useTheme: jest.fn(() => "light"),
}));

const consoleErrorSpy = jest.spyOn(console, "error");
const { useTheme } = jest.requireMock("@/app/store/themeStore");

describe("LazyCodeEditor", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useTheme.mockReturnValue("light");
  });

  afterAll(() => {
    consoleErrorSpy.mockRestore();
  });

  test("renders with value prop", async () => {
    const onChange = jest.fn();
    const value = '{"key": "value"}';

    const { container, getByTestId } = render(
      <LazyCodeEditor value={value} onChange={onChange} />
    );

    await waitFor(() => {
      expect(getByTestId("codemirror")).toBeInTheDocument();
    });
    expect(container).toBeInTheDocument();
  });

  test("renders with onChange handler", async () => {
    const onChange = jest.fn();

    const { container, getByTestId } = render(
      <LazyCodeEditor value="" onChange={onChange} />
    );

    await waitFor(() => {
      expect(getByTestId("codemirror")).toBeInTheDocument();
    });
    expect(container).toBeInTheDocument();
  });

  test("renders with custom height", async () => {
    const onChange = jest.fn();

    const { container, getByTestId } = render(
      <LazyCodeEditor value="" onChange={onChange} height="500px" />
    );

    await waitFor(() => {
      expect(getByTestId("codemirror")).toBeInTheDocument();
    });
    expect(container).toBeInTheDocument();
  });

  test("uses light theme when app theme is light", async () => {
    const onChange = jest.fn();

    const { getByTestId } = render(
      <LazyCodeEditor value="{}" onChange={onChange} />
    );

    await waitFor(() => {
      expect(getByTestId("codemirror")).toHaveAttribute("data-theme", "light");
    });
  });

  test("uses dark theme when app theme is dark", async () => {
    useTheme.mockReturnValue("dark");
    const onChange = jest.fn();

    const { getByTestId } = render(
      <LazyCodeEditor value="{}" onChange={onChange} />
    );

    await waitFor(() => {
      expect(getByTestId("codemirror")).toHaveAttribute("data-theme", "dark");
    });
  });
});
