require("@testing-library/jest-dom");

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.ResizeObserver = ResizeObserverMock;

const baseMessages = require("./app/i18n/en-US.json");
const commonMessages = require("./app/i18n/common.json");
const messages = { ...commonMessages, ...baseMessages };

const interpolate = (template, vars) => {
  if (!vars) {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (_, key) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) {
      return String(vars[key]);
    }
    return `{${key}}`;
  });
};

jest.mock("@/app/i18n", () => ({
  I18nProvider: ({ children }) => children,
  useT: () => (key, vars) => interpolate(messages[key] || key, vars),
}));
