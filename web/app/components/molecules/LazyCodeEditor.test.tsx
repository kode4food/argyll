import React from "react";
import { render } from "@testing-library/react";
import LazyCodeEditor from "./LazyCodeEditor";

jest.mock("@uiw/react-codemirror", () => ({
  __esModule: true,
  default: jest.fn(() => <div data-testid="codemirror" />),
}));

const consoleErrorSpy = jest.spyOn(console, "error");

describe("LazyCodeEditor", () => {
  afterAll(() => {
    consoleErrorSpy.mockRestore();
  });

  test("renders with value prop", () => {
    const onChange = jest.fn();
    const value = '{"key": "value"}';

    const { container } = render(
      <LazyCodeEditor value={value} onChange={onChange} />
    );

    expect(container).toBeInTheDocument();
  });

  test("renders with onChange handler", () => {
    const onChange = jest.fn();

    const { container } = render(
      <LazyCodeEditor value="" onChange={onChange} />
    );

    expect(container).toBeInTheDocument();
  });

  test("renders with custom height", () => {
    const onChange = jest.fn();

    const { container } = render(
      <LazyCodeEditor value="" onChange={onChange} height="500px" />
    );

    expect(container).toBeInTheDocument();
  });
});
