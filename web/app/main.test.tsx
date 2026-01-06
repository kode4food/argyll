import React from "react";
import App from "./App";

let createRootMock: jest.Mock;
let renderMock: jest.Mock;

jest.mock("react-dom/client", () => ({
  __esModule: true,
  default: {
    createRoot: (...args: unknown[]) => createRootMock(...args),
  },
}));

jest.mock("./App", () => ({
  __esModule: true,
  default: () => <div data-testid="app" />,
}));

describe("main", () => {
  beforeEach(() => {
    renderMock = jest.fn();
    createRootMock = jest.fn(() => ({ render: renderMock }));
    document.body.innerHTML = '<div id="root"></div>';
  });

  afterEach(() => {
    jest.resetModules();
  });

  test("mounts the app into the root element", () => {
    jest.isolateModules(() => {
      require("./main");
    });

    const rootEl = document.getElementById("root");

    expect(createRootMock).toHaveBeenCalledWith(rootEl);
    expect(renderMock).toHaveBeenCalledTimes(1);

    const renderedElement = renderMock.mock.calls[0][0];
    expect(renderedElement.type).toBe(React.StrictMode);
    expect(renderedElement.props.children.type).toBe(App);
  });
});
